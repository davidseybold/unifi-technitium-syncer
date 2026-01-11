package syncer

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	"github.com/davidseybold/unifi-technitium-sync/technitium"
	"github.com/davidseybold/unifi-technitium-sync/unifi"
)

type Syncer struct {
	uc     *unifi.Client
	tc     *technitium.Client
	zone   string
	logger *slog.Logger
}

func New(uc *unifi.Client, tc *technitium.Client, syncZone string, logger *slog.Logger) *Syncer {
	return &Syncer{
		uc:     uc,
		tc:     tc,
		zone:   syncZone,
		logger: logger,
	}
}

func (s *Syncer) Run(ctx context.Context) error {

	if s.zone == "" {
		return fmt.Errorf("zone is required")
	}

	if s.logger == nil {
		s.logger = slog.New(slog.DiscardHandler)
	}

	_, err := s.tc.GetZone(ctx, s.zone)
	if err != nil {
		return fmt.Errorf("failed to validate sync zone exists: %v", err)
	}

	existingRecords, err := s.getExistingRecords(ctx)
	if err != nil {
		return fmt.Errorf("failed to get existing records: %v", err)
	}

	clientRecords, err := s.getUnifiRecords(ctx)
	if err != nil {
		return fmt.Errorf("failed to get unifi records: %v", err)
	}

	addRecords, deleteRecords := s.calculateChanges(existingRecords, clientRecords)

	for _, r := range deleteRecords {
		err := s.tc.DeleteRecord(ctx, s.zone, r.Name, r.Type, *r.RData.IPAddress)
		if err != nil {
			s.logger.Error("error deleting record", "record", r.Name, "error", err)
		}
		s.logger.Info("deleted record", "record", r.Name)
	}

	for _, r := range addRecords {
		err := s.tc.AddRecord(ctx, s.zone, r.Name, *r.RData.IPAddress, r.TTL, r.Comments)
		if err != nil {
			s.logger.Error("error upserting record", "record", r.Name, "error", err)
		}
		s.logger.Info("upserted record", "record", r.Name)
	}

	return nil
}

func (s *Syncer) getUnifiRecords(ctx context.Context) (map[string]technitium.Record, error) {
	unifiClients, err := s.uc.ListClients(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list clients: %v", err)
	}

	clientRecords := map[string]technitium.Record{}
	for _, client := range unifiClients {
		sanitizedUnifiName := sanitizeDNS(client.Name)
		name := fmt.Sprintf("%s.%s", sanitizedUnifiName, s.zone)

		clientRecords[name] = technitium.Record{
			Name:     name,
			Type:     "A",
			TTL:      3600,
			Comments: client.MacAddress,
			RData:    technitium.RData{IPAddress: &client.IPAddress},
		}
	}

	return clientRecords, nil
}

func (s *Syncer) getExistingRecords(ctx context.Context) ([]technitium.Record, error) {
	records, err := s.tc.ListRecords(ctx, s.zone)
	if err != nil {
		return nil, fmt.Errorf("failed to list existing records: %v", err)
	}

	aRecords := filterTechnitiumRecordsByType(records, "A")

	return aRecords, nil
}

func (s *Syncer) calculateChanges(existingRecords []technitium.Record, clientRecords map[string]technitium.Record) ([]technitium.Record, []technitium.Record) {
	addRecords := []technitium.Record{}
	deleteRecords := []technitium.Record{}

	for _, record := range existingRecords {
		clientRecord, ok := clientRecords[record.Name]
		if !ok {
			deleteRecords = append(deleteRecords, record)
		} else if !shouldUpdateRecord(record, clientRecord) {
			s.logger.Info("record is up to date", "record", record.Name)
			delete(clientRecords, record.Name)
		}
	}

	for _, cr := range clientRecords {
		addRecords = append(addRecords, cr)
	}

	return addRecords, deleteRecords
}

func filterTechnitiumRecordsByType(records []technitium.Record, typ string) []technitium.Record {
	var filtered []technitium.Record
	for _, record := range records {
		if record.Type == typ {
			filtered = append(filtered, record)
		}
	}

	return filtered
}

func shouldUpdateRecord(existingRecord technitium.Record, clientRecord technitium.Record) bool {
	if existingRecord.RData.IPAddress == nil || clientRecord.RData.IPAddress == nil {
		return true
	}
	return *existingRecord.RData.IPAddress != *clientRecord.RData.IPAddress
}

func sanitizeDNS(input string) string {
	s := strings.ToLower(input)
	s = strings.ReplaceAll(s, "'", "")
	reg := regexp.MustCompile(`[^a-z0-9]+`)
	s = reg.ReplaceAllString(s, "-")

	s = strings.Trim(s, "-")
	if len(s) > 63 {
		s = s[:63]
		s = strings.Trim(s, "-")
	}

	return s
}
