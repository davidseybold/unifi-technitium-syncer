package syncer

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/davidseybold/unifi-dns-sync/dnsprovider"
	"github.com/davidseybold/unifi-dns-sync/unifi"
)

type DNSProvider interface {
	GetZone(ctx context.Context, zone string) (dnsprovider.Zone, error)
	ListRecords(ctx context.Context, zone string) ([]dnsprovider.Record, error)
	UpsertRecord(ctx context.Context, zone string, record dnsprovider.Record) error
	DeleteRecord(ctx context.Context, zone string, record dnsprovider.Record) error
}

type Syncer struct {
	uc       *unifi.Client
	provider DNSProvider
	config   Config
	logger   *slog.Logger
}

type Config struct {
	SyncZone       string
	ClientWaitTime time.Duration
	StateDir       string
}

func New(uc *unifi.Client, provider DNSProvider, config Config, logger *slog.Logger) *Syncer {
	return &Syncer{
		uc:       uc,
		provider: provider,
		config:   config,
		logger:   logger,
	}
}

func (s *Syncer) Run(ctx context.Context) (*SyncResult, error) {

	if s.config.SyncZone == "" {
		return nil, fmt.Errorf("zone is required")
	}

	if s.logger == nil {
		s.logger = slog.New(slog.DiscardHandler)
	}

	_, err := s.provider.GetZone(ctx, s.config.SyncZone)
	if err != nil {
		return nil, fmt.Errorf("failed to validate sync zone exists: %v", err)
	}

	state, err := s.loadState()
	if err != nil {
		return nil, fmt.Errorf("failed to load state: %v", err)
	}

	unifiClients, err := s.uc.ListConnectedClients(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list clients: %v", err)
	}

	newState, err := s.updateState(state, unifiClients)
	if err != nil {
		return nil, fmt.Errorf("failed to update state: %v", err)
	}

	existingRecords, err := s.getExistingRecords(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get existing records: %v", err)
	}

	clientRecords, err := s.convertClientsToRecords(ctx, newState.Clients)
	if err != nil {
		return nil, fmt.Errorf("failed to get unifi records: %v", err)
	}

	changes := s.calculateChanges(existingRecords, clientRecords)

	result := s.processChanges(ctx, changes)

	err = s.persistState(newState)
	if err != nil {
		s.logger.Error("failed to persist state", "error", err)
	}

	return &result, nil
}

type changes struct {
	Add    []dnsprovider.Record
	Delete []dnsprovider.Record
}

type SyncResult struct {
	AddSuccess int
	AddFailed  int

	DeleteSuccess int
	DeleteFailed  int
}

type state struct {
	Clients []client `json:"clients"`
}

func (s *state) GetClientMacLookup() map[string]client {
	clients := map[string]client{}
	for _, client := range s.Clients {
		clients[client.MACAddress] = client
	}
	return clients
}

type client struct {
	Name       string    `json:"name"`
	MACAddress string    `json:"macAddress"`
	IPAddress  string    `json:"ipAddress"`
	LastSeen   time.Time `json:"lastSeen"`
}

func (s *Syncer) loadState() (*state, error) {

	if _, err := os.Stat(s.getStateFileName()); os.IsNotExist(err) {
		return &state{Clients: []client{}}, nil
	}

	data, err := os.ReadFile(s.getStateFileName())
	if err != nil {
		return nil, fmt.Errorf("failed to read state file: %v", err)
	}

	var state state
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal state file: %v", err)
	}

	return &state, nil
}

func (s *Syncer) persistState(state *state) error {
	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("failed to marshal state: %v", err)
	}

	return os.WriteFile(s.getStateFileName(), data, 0644)
}

func (s *Syncer) getStateFileName() string {
	return s.config.StateDir + "/state.json"
}

func (s *Syncer) convertClientsToRecords(ctx context.Context, clients []client) (map[string]dnsprovider.Record, error) {
	clientRecords := map[string]dnsprovider.Record{}
	for _, client := range clients {
		sanitizedUnifiName := sanitizeDNS(client.Name)
		name := fmt.Sprintf("%s.%s", sanitizedUnifiName, s.config.SyncZone)

		clientRecords[name] = dnsprovider.Record{
			Name:      name,
			Type:      "A",
			TTL:       3600,
			Comments:  client.MACAddress,
			IPAddress: client.IPAddress,
		}
	}

	return clientRecords, nil
}

func (s *Syncer) updateState(currentState *state, currentClients []unifi.NetworkClient) (*state, error) {
	newClientMap := map[string]client{}

	for _, c := range currentClients {
		newClientMap[c.MacAddress] = client{
			Name:       c.Name,
			MACAddress: c.MacAddress,
			IPAddress:  c.IPAddress,
			LastSeen:   time.Now(),
		}
	}

	for _, c := range currentState.Clients {
		lastSeenDiff := time.Since(c.LastSeen)
		if lastSeenDiff > s.config.ClientWaitTime {
			continue
		}

		if _, ok := newClientMap[c.MACAddress]; !ok {
			newClientMap[c.MACAddress] = c
		}
	}

	newClients := []client{}
	for _, c := range newClientMap {
		newClients = append(newClients, c)
	}

	return &state{Clients: newClients}, nil
}

func (s *Syncer) getExistingRecords(ctx context.Context) ([]dnsprovider.Record, error) {
	records, err := s.provider.ListRecords(ctx, s.config.SyncZone)
	if err != nil {
		return nil, fmt.Errorf("failed to list existing records: %v", err)
	}

	aRecords := filterTechnitiumRecordsByType(records, "A")

	return aRecords, nil
}

func (s *Syncer) calculateChanges(existingRecords []dnsprovider.Record, clientRecords map[string]dnsprovider.Record) changes {
	addRecords := []dnsprovider.Record{}
	deleteRecords := []dnsprovider.Record{}

	for _, record := range existingRecords {
		clientRecord, ok := clientRecords[record.Name]
		if !ok {
			deleteRecords = append(deleteRecords, record)
		} else if !shouldUpdateRecord(record, clientRecord) {
			s.logger.Debug("record is up to date", "record", record.Name)
			delete(clientRecords, record.Name)
		}
	}

	for _, cr := range clientRecords {
		addRecords = append(addRecords, cr)
	}

	return changes{Add: addRecords, Delete: deleteRecords}
}

func (s *Syncer) processChanges(ctx context.Context, c changes) SyncResult {
	result := SyncResult{}

	for _, r := range c.Delete {
		err := s.provider.DeleteRecord(ctx, s.config.SyncZone, r)
		if err != nil {
			s.logger.Error("error deleting record", "record", r.Name, "error", err)
			result.DeleteFailed++
		} else {
			result.DeleteSuccess++
			s.logger.Debug("deleted record", "record", r.Name)
		}
	}

	for _, r := range c.Add {
		err := s.provider.UpsertRecord(ctx, s.config.SyncZone, r)
		if err != nil {
			s.logger.Error("error upserting record", "record", r.Name, "error", err)
			result.AddFailed++
		} else {
			result.AddSuccess++
			s.logger.Debug("upserted record", "record", r.Name)
		}
	}

	return result
}

func filterTechnitiumRecordsByType(records []dnsprovider.Record, typ string) []dnsprovider.Record {
	var filtered []dnsprovider.Record
	for _, record := range records {
		if record.Type == typ {
			filtered = append(filtered, record)
		}
	}

	return filtered
}

func shouldUpdateRecord(existingRecord dnsprovider.Record, clientRecord dnsprovider.Record) bool {
	if existingRecord.IPAddress == "" || clientRecord.IPAddress == "" {
		return true
	}
	return existingRecord.IPAddress != clientRecord.IPAddress
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
