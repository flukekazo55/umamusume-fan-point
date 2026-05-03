package mongodb

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"umamusume-fan-point/backend/internal/excel"
	"umamusume-fan-point/backend/internal/persistence"
)

type Store struct {
	client  *mongo.Client
	months  *mongo.Collection
	players *mongo.Collection
}

type monthDoc struct {
	ID        string    `bson:"_id"`
	Label     string    `bson:"label"`
	StartDate string    `bson:"start_date"`
	EndDate   string    `bson:"end_date"`
	Dates     []string  `bson:"dates"`
	CreatedAt time.Time `bson:"created_at"`
	UpdatedAt time.Time `bson:"updated_at"`
}

type playerDoc struct {
	ID        string           `bson:"_id"`
	MonthID   string           `bson:"month_id"`
	NameKey   string           `bson:"name_key"`
	Name      string           `bson:"name"`
	Debt      int64            `bson:"debt"`
	Note      string           `bson:"note"`
	Snapshots []excel.Snapshot `bson:"snapshots"`
	CreatedAt time.Time        `bson:"created_at"`
	UpdatedAt time.Time        `bson:"updated_at"`
}

func New(ctx context.Context, uri string, database string) (*Store, error) {
	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		return nil, fmt.Errorf("connect mongo: %w", err)
	}
	if err := client.Ping(ctx, nil); err != nil {
		_ = client.Disconnect(ctx)
		return nil, fmt.Errorf("ping mongo: %w", err)
	}

	db := client.Database(database)
	store := &Store{
		client:  client,
		months:  db.Collection("months"),
		players: db.Collection("players"),
	}
	if err := store.ensureIndexes(ctx); err != nil {
		_ = client.Disconnect(ctx)
		return nil, err
	}
	return store, nil
}

func (s *Store) Close(ctx context.Context) error {
	return s.client.Disconnect(ctx)
}

func (s *Store) SeedIfEmpty(ctx context.Context, loader interface {
	Load() (*excel.Workbook, error)
}) error {
	count, err := s.months.CountDocuments(ctx, bson.D{})
	if err != nil {
		return fmt.Errorf("count seeded months: %w", err)
	}
	if count > 0 {
		return nil
	}

	workbook, err := loader.Load()
	if err != nil {
		return fmt.Errorf("load seed workbook: %w", err)
	}

	now := time.Now().UTC()
	monthWrites := make([]mongo.WriteModel, 0, len(workbook.Months))
	playerWrites := make([]mongo.WriteModel, 0)
	for _, month := range workbook.Months {
		monthWrites = append(monthWrites, mongo.NewInsertOneModel().SetDocument(monthDoc{
			ID:        month.ID,
			Label:     month.Label,
			StartDate: month.StartDate,
			EndDate:   month.EndDate,
			Dates:     month.Dates,
			CreatedAt: now,
			UpdatedAt: now,
		}))

		for _, member := range month.Members {
			doc := playerDoc{
				ID:        playerID(month.ID, member.Name),
				MonthID:   month.ID,
				NameKey:   nameKey(member.Name),
				Name:      member.Name,
				Debt:      member.Debt,
				Note:      member.Note,
				Snapshots: compactSnapshots(member.Snapshots),
				CreatedAt: now,
				UpdatedAt: now,
			}
			playerWrites = append(playerWrites, mongo.NewInsertOneModel().SetDocument(doc))
		}
	}

	if len(monthWrites) > 0 {
		if _, err := s.months.BulkWrite(ctx, monthWrites); err != nil {
			return fmt.Errorf("seed months: %w", err)
		}
	}
	if len(playerWrites) > 0 {
		if _, err := s.players.BulkWrite(ctx, playerWrites); err != nil {
			return fmt.Errorf("seed players: %w", err)
		}
	}
	return nil
}

func (s *Store) Load() (*excel.Workbook, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	months, err := s.loadMonths(ctx)
	if err != nil {
		return nil, err
	}

	seenMembers := map[string]struct{}{}
	var totalCurrent int64
	for index := range months {
		players, err := s.playerDocs(ctx, months[index].ID)
		if err != nil {
			return nil, err
		}
		months[index].Members = membersFromDocs(months[index].Dates, players)
		months[index] = excel.NormalizeMonth(months[index])
		for _, member := range months[index].Members {
			seenMembers[member.Name] = struct{}{}
		}
		if index == 0 {
			totalCurrent = months[index].Stats.TotalCurrent
		}
	}

	latestID := ""
	if len(months) > 0 {
		latestID = months[0].ID
	}

	return &excel.Workbook{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		LatestID:    latestID,
		Months:      months,
		Summary: excel.BookSummary{
			MonthCount:       len(months),
			TrackedMembers:   len(seenMembers),
			TotalCurrentFans: totalCurrent,
		},
	}, nil
}

func (s *Store) ListPlayers(ctx context.Context, monthID string) ([]excel.Member, error) {
	month, err := s.monthDoc(ctx, monthID)
	if err != nil {
		return nil, err
	}
	players, err := s.playerDocs(ctx, monthID)
	if err != nil {
		return nil, err
	}
	month.Members = membersFromDocs(month.Dates, players)
	month = excel.NormalizeMonth(month)
	return month.Members, nil
}

func (s *Store) CreateMonth(ctx context.Context, input excel.MonthInput) (*excel.Month, error) {
	month, err := monthFromInput(input, "")
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	doc := monthDoc{
		ID:        month.ID,
		Label:     month.Label,
		StartDate: month.StartDate,
		EndDate:   month.EndDate,
		Dates:     month.Dates,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if _, err := s.months.InsertOne(ctx, doc); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return nil, persistence.ErrConflict
		}
		return nil, fmt.Errorf("create month: %w", err)
	}

	if strings.TrimSpace(input.SourceMonthID) != "" {
		if err := s.clonePlayers(ctx, input.SourceMonthID, month.ID, month.Dates); err != nil {
			_, _ = s.months.DeleteOne(ctx, bson.D{{Key: "_id", Value: month.ID}})
			return nil, err
		}
	}
	return s.GetMonth(ctx, month.ID)
}

func (s *Store) GetMonth(ctx context.Context, monthID string) (*excel.Month, error) {
	month, err := s.monthDoc(ctx, monthID)
	if err != nil {
		return nil, err
	}
	players, err := s.playerDocs(ctx, monthID)
	if err != nil {
		return nil, err
	}
	month.Members = membersFromDocs(month.Dates, players)
	month = excel.NormalizeMonth(month)
	return &month, nil
}

func (s *Store) UpdateMonth(ctx context.Context, monthID string, input excel.MonthInput) (*excel.Month, error) {
	month, err := monthFromInput(input, monthID)
	if err != nil {
		return nil, err
	}

	result, err := s.months.UpdateOne(ctx, bson.D{{Key: "_id", Value: monthID}}, bson.D{{Key: "$set", Value: bson.D{
		{Key: "label", Value: month.Label},
		{Key: "start_date", Value: month.StartDate},
		{Key: "end_date", Value: month.EndDate},
		{Key: "dates", Value: month.Dates},
		{Key: "updated_at", Value: time.Now().UTC()},
	}}})
	if err != nil {
		return nil, fmt.Errorf("update month: %w", err)
	}
	if result.MatchedCount == 0 {
		return nil, persistence.ErrNotFound
	}

	if err := s.standardizePlayers(ctx, monthID, month.Dates); err != nil {
		return nil, err
	}
	return s.GetMonth(ctx, monthID)
}

func (s *Store) DeleteMonth(ctx context.Context, monthID string) error {
	result, err := s.months.DeleteOne(ctx, bson.D{{Key: "_id", Value: monthID}})
	if err != nil {
		return fmt.Errorf("delete month: %w", err)
	}
	if result.DeletedCount == 0 {
		return persistence.ErrNotFound
	}
	if _, err := s.players.DeleteMany(ctx, bson.D{{Key: "month_id", Value: monthID}}); err != nil {
		return fmt.Errorf("delete month players: %w", err)
	}
	return nil
}

func (s *Store) GetPlayer(ctx context.Context, monthID string, name string) (*excel.Member, error) {
	month, err := s.monthDoc(ctx, monthID)
	if err != nil {
		return nil, err
	}
	players, err := s.playerDocs(ctx, monthID)
	if err != nil {
		return nil, err
	}
	month.Members = membersFromDocs(month.Dates, players)
	month = excel.NormalizeMonth(month)
	key := nameKey(name)
	for _, member := range month.Members {
		if nameKey(member.Name) == key {
			return &member, nil
		}
	}
	return nil, persistence.ErrNotFound
}

func (s *Store) CreatePlayer(ctx context.Context, monthID string, input excel.PlayerInput) (*excel.Member, error) {
	month, err := s.monthDoc(ctx, monthID)
	if err != nil {
		return nil, err
	}

	member, ok := excel.BuildMember(input, month.Dates)
	if !ok {
		return nil, fmt.Errorf("player requires a name and at least one valid snapshot")
	}

	now := time.Now().UTC()
	doc := playerDoc{
		ID:        playerID(monthID, member.Name),
		MonthID:   monthID,
		NameKey:   nameKey(member.Name),
		Name:      member.Name,
		Debt:      member.Debt,
		Note:      member.Note,
		Snapshots: compactSnapshots(input.Snapshots),
		CreatedAt: now,
		UpdatedAt: now,
	}
	if _, err := s.players.InsertOne(ctx, doc); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return nil, persistence.ErrConflict
		}
		return nil, fmt.Errorf("create player: %w", err)
	}
	return s.GetPlayer(ctx, monthID, member.Name)
}

func (s *Store) UpdatePlayer(ctx context.Context, monthID string, name string, input excel.PlayerInput) (*excel.Member, error) {
	month, err := s.monthDoc(ctx, monthID)
	if err != nil {
		return nil, err
	}

	if strings.TrimSpace(input.Name) == "" {
		input.Name = name
	}
	member, ok := excel.BuildMember(input, month.Dates)
	if !ok {
		return nil, fmt.Errorf("player requires a name and at least one valid snapshot")
	}

	update := bson.D{{Key: "$set", Value: bson.D{
		{Key: "name", Value: member.Name},
		{Key: "name_key", Value: nameKey(member.Name)},
		{Key: "debt", Value: member.Debt},
		{Key: "note", Value: member.Note},
		{Key: "snapshots", Value: compactSnapshots(input.Snapshots)},
		{Key: "updated_at", Value: time.Now().UTC()},
	}}}
	result, err := s.players.UpdateOne(ctx, bson.D{
		{Key: "month_id", Value: monthID},
		{Key: "name_key", Value: nameKey(name)},
	}, update)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return nil, persistence.ErrConflict
		}
		return nil, fmt.Errorf("update player: %w", err)
	}
	if result.MatchedCount == 0 {
		return nil, persistence.ErrNotFound
	}
	return s.GetPlayer(ctx, monthID, member.Name)
}

func (s *Store) DeletePlayer(ctx context.Context, monthID string, name string) error {
	result, err := s.players.DeleteOne(ctx, bson.D{
		{Key: "month_id", Value: monthID},
		{Key: "name_key", Value: nameKey(name)},
	})
	if err != nil {
		return fmt.Errorf("delete player: %w", err)
	}
	if result.DeletedCount == 0 {
		return persistence.ErrNotFound
	}
	return nil
}

func (s *Store) ensureIndexes(ctx context.Context) error {
	_, err := s.players.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "month_id", Value: 1}},
			Options: options.Index().SetName("players_month_id"),
		},
		{
			Keys: bson.D{
				{Key: "month_id", Value: 1},
				{Key: "name_key", Value: 1},
			},
			Options: options.Index().SetName("players_month_name_unique").SetUnique(true),
		},
	})
	if err != nil {
		return fmt.Errorf("create player indexes: %w", err)
	}
	return nil
}

func (s *Store) loadMonths(ctx context.Context) ([]excel.Month, error) {
	cursor, err := s.months.Find(ctx, bson.D{}, options.Find().SetSort(bson.D{{Key: "start_date", Value: -1}}))
	if err != nil {
		return nil, fmt.Errorf("find months: %w", err)
	}
	defer cursor.Close(ctx)

	months := make([]excel.Month, 0)
	for cursor.Next(ctx) {
		var doc monthDoc
		if err := cursor.Decode(&doc); err != nil {
			return nil, fmt.Errorf("decode month: %w", err)
		}
		months = append(months, doc.month())
	}
	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("iterate months: %w", err)
	}
	return months, nil
}

func (s *Store) monthDoc(ctx context.Context, monthID string) (excel.Month, error) {
	var doc monthDoc
	err := s.months.FindOne(ctx, bson.D{{Key: "_id", Value: monthID}}).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return excel.Month{}, persistence.ErrNotFound
	}
	if err != nil {
		return excel.Month{}, fmt.Errorf("find month: %w", err)
	}
	return doc.month(), nil
}

func (s *Store) playerDocs(ctx context.Context, monthID string) ([]playerDoc, error) {
	cursor, err := s.players.Find(ctx, bson.D{{Key: "month_id", Value: monthID}}, options.Find().SetSort(bson.D{{Key: "name_key", Value: 1}}))
	if err != nil {
		return nil, fmt.Errorf("find players: %w", err)
	}
	defer cursor.Close(ctx)

	players := make([]playerDoc, 0)
	for cursor.Next(ctx) {
		var doc playerDoc
		if err := cursor.Decode(&doc); err != nil {
			return nil, fmt.Errorf("decode player: %w", err)
		}
		players = append(players, doc)
	}
	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("iterate players: %w", err)
	}
	return players, nil
}

func (d monthDoc) month() excel.Month {
	return excel.Month{
		ID:        d.ID,
		Label:     d.Label,
		StartDate: d.StartDate,
		EndDate:   d.EndDate,
		Dates:     d.Dates,
	}
}

func membersFromDocs(dates []string, players []playerDoc) []excel.Member {
	members := make([]excel.Member, 0, len(players))
	for _, doc := range players {
		member, ok := excel.BuildMember(excel.PlayerInput{
			Name:      doc.Name,
			Debt:      doc.Debt,
			Note:      doc.Note,
			Snapshots: doc.Snapshots,
		}, dates)
		if ok {
			members = append(members, member)
		}
	}
	return members
}

func (s *Store) clonePlayers(ctx context.Context, sourceMonthID string, targetMonthID string, targetDates []string) error {
	sourceMonth, err := s.GetMonth(ctx, sourceMonthID)
	if err != nil {
		return err
	}
	if len(targetDates) == 0 {
		return fmt.Errorf("target month requires at least one date")
	}

	now := time.Now().UTC()
	writes := make([]mongo.WriteModel, 0, len(sourceMonth.Members))
	for _, member := range sourceMonth.Members {
		doc := playerDoc{
			ID:        playerID(targetMonthID, member.Name),
			MonthID:   targetMonthID,
			NameKey:   nameKey(member.Name),
			Name:      member.Name,
			Debt:      member.Debt,
			Note:      member.Note,
			Snapshots: []excel.Snapshot{{Date: targetDates[0], Fans: member.CurrentFans}},
			CreatedAt: now,
			UpdatedAt: now,
		}
		writes = append(writes, mongo.NewInsertOneModel().SetDocument(doc))
	}
	if len(writes) == 0 {
		return nil
	}
	if _, err := s.players.BulkWrite(ctx, writes); err != nil {
		return fmt.Errorf("clone source month players: %w", err)
	}
	return nil
}

func (s *Store) standardizePlayers(ctx context.Context, monthID string, dates []string) error {
	players, err := s.playerDocs(ctx, monthID)
	if err != nil {
		return err
	}
	writes := make([]mongo.WriteModel, 0, len(players))
	for _, doc := range players {
		member, ok := excel.BuildMember(excel.PlayerInput{
			Name:      doc.Name,
			Debt:      doc.Debt,
			Note:      doc.Note,
			Snapshots: doc.Snapshots,
		}, dates)
		if !ok && len(dates) > 0 {
			fans := latestFans(doc.Snapshots)
			if fans > 0 {
				member, ok = excel.BuildMember(excel.PlayerInput{
					Name:      doc.Name,
					Debt:      doc.Debt,
					Note:      doc.Note,
					Snapshots: []excel.Snapshot{{Date: dates[0], Fans: fans}},
				}, dates)
			}
		}
		if !ok {
			continue
		}
		writes = append(writes, mongo.NewUpdateOneModel().
			SetFilter(bson.D{{Key: "_id", Value: doc.ID}}).
			SetUpdate(bson.D{{Key: "$set", Value: bson.D{
				{Key: "snapshots", Value: compactSnapshots(member.Snapshots)},
				{Key: "updated_at", Value: time.Now().UTC()},
			}}}))
	}
	if len(writes) == 0 {
		return nil
	}
	if _, err := s.players.BulkWrite(ctx, writes); err != nil {
		return fmt.Errorf("standardize month players: %w", err)
	}
	return nil
}

func monthFromInput(input excel.MonthInput, fallbackID string) (excel.Month, error) {
	id := strings.TrimSpace(input.ID)
	if id == "" {
		id = strings.TrimSpace(fallbackID)
	}
	label := strings.TrimSpace(input.Label)
	startDate := strings.TrimSpace(input.StartDate)
	endDate := strings.TrimSpace(input.EndDate)
	dates := cleanDates(input.Dates)
	if len(dates) == 0 {
		return excel.Month{}, fmt.Errorf("month requires at least one date")
	}
	if id == "" {
		id = strings.ReplaceAll(dates[0], "-", "")
	}
	if label == "" {
		label = id
	}
	if startDate == "" {
		startDate = dates[0]
	}
	if endDate == "" {
		endDate = dates[len(dates)-1]
	}
	return excel.Month{
		ID:        id,
		Label:     label,
		StartDate: startDate,
		EndDate:   endDate,
		Dates:     dates,
	}, nil
}

func cleanDates(values []string) []string {
	seen := map[string]struct{}{}
	dates := make([]string, 0, len(values))
	for _, value := range values {
		date := strings.TrimSpace(value)
		if date == "" {
			continue
		}
		if _, ok := seen[date]; ok {
			continue
		}
		seen[date] = struct{}{}
		dates = append(dates, date)
	}
	sort.Strings(dates)
	return dates
}

func latestFans(snapshots []excel.Snapshot) int64 {
	var latest string
	var fans int64
	for _, snapshot := range snapshots {
		if snapshot.Fans <= 0 {
			continue
		}
		if snapshot.Date >= latest {
			latest = snapshot.Date
			fans = snapshot.Fans
		}
	}
	return fans
}

func compactSnapshots(snapshots []excel.Snapshot) []excel.Snapshot {
	result := make([]excel.Snapshot, 0, len(snapshots))
	for _, snapshot := range snapshots {
		if snapshot.Fans > 0 {
			result = append(result, snapshot)
		}
	}
	sort.SliceStable(result, func(i, j int) bool {
		return result[i].Date < result[j].Date
	})
	compacted := result[:0]
	var previousFans int64
	for index, snapshot := range result {
		if index == 0 || snapshot.Fans != previousFans {
			compacted = append(compacted, snapshot)
		}
		previousFans = snapshot.Fans
	}
	return compacted
}

func playerID(monthID string, name string) string {
	return monthID + ":" + nameKey(name)
}

func nameKey(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}
