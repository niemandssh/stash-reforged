package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
	"github.com/jmoiron/sqlx"
	"github.com/stashapp/stash/pkg/models"
	"gopkg.in/guregu/null.v4"
	"gopkg.in/guregu/null.v4/zero"
)

const (
	gameTable           = "games"
	gameIDColumn        = "game_id"
	gamesODatesTable    = "games_o_dates"
	gameODateColumn     = "o_date"
	gamesOMGDatesTable  = "games_omg_dates"
	gameOMGDateColumn   = "omg_date"
	gamesViewDatesTable = "games_view_dates"
	gameViewDateColumn  = "view_date"
	gamesTagsTable      = "games_tags"
	gamesURLsTable      = "game_urls"
	gameURLColumn       = "url"
)

type gameRow struct {
	ID             int         `db:"id" goqu:"skipinsert"`
	Title          zero.String `db:"title"`
	Details        zero.String `db:"details"`
	Date           NullDate    `db:"date"`
	Rating         null.Int    `db:"rating"`
	Organized      bool        `db:"organized"`
	OCounter       int         `db:"o_counter"`
	OmegCounter    int         `db:"omg_counter"`
	Image          []byte      `db:"image"`
	FolderPath     zero.String `db:"folder_path"`
	ExecutablePath zero.String `db:"executable_path"`
	CreatedAt      Timestamp   `db:"created_at"`
	UpdatedAt      Timestamp   `db:"updated_at"`
}

func (r *gameRow) fromGame(g models.Game) {
	r.ID = g.ID
	r.Title = zero.StringFrom(g.Title)
	r.Details = zero.StringFrom(g.Details)
	r.Date = NullDateFromDatePtr(g.Date)
	r.Rating = intFromPtr(g.Rating)
	r.Organized = g.Organized
	r.OCounter = g.OCounter
	r.OmegCounter = g.OmegCounter
	r.Image = g.Image
	r.FolderPath = zero.StringFrom(g.FolderPath)
	r.ExecutablePath = zero.StringFrom(g.ExecutablePath)
	r.CreatedAt = Timestamp{Timestamp: g.CreatedAt}
	r.UpdatedAt = Timestamp{Timestamp: g.UpdatedAt}
}

func (r *gameRow) resolve() *models.Game {
	return &models.Game{
		ID:             r.ID,
		Title:          r.Title.String,
		Details:        r.Details.String,
		Date:           r.Date.DatePtr(),
		Rating:         nullIntPtr(r.Rating),
		Organized:      r.Organized,
		OCounter:       r.OCounter,
		OmegCounter:    r.OmegCounter,
		Image:          r.Image,
		FolderPath:     r.FolderPath.String,
		ExecutablePath: r.ExecutablePath.String,
		CreatedAt:      r.CreatedAt.Timestamp,
		UpdatedAt:      r.UpdatedAt.Timestamp,
	}
}

type gameRowRecord struct {
	updateRecord
}

func (r *gameRowRecord) fromPartial(p models.GamePartial) {
	r.setString("title", p.Title)
	r.setNullString("details", p.Details)
	r.setNullDate("date", p.Date)
	r.setNullInt("rating", p.Rating)
	r.setBool("organized", p.Organized)
	r.setInt("o_counter", p.OCounter)
	r.setInt("omg_counter", p.OmegCounter)
	if p.Image.Set {
		if p.Image.Null {
			r.set("image", nil)
		} else {
			r.set("image", p.Image.Value)
		}
	}
	r.setString("folder_path", p.FolderPath)
	r.setString("executable_path", p.ExecutablePath)
	r.setTimestamp("created_at", p.CreatedAt)
	r.setTimestamp("updated_at", p.UpdatedAt)
}

type gameRepositoryType struct {
	repository
}

var gameRepository = gameRepositoryType{
	repository: repository{
		tableName: gameTable,
		idColumn:  idColumn,
	},
}

type GameStore struct {
	tableMgr        *table
	oDateManager    oDateManager
	omgDateManager  omgDateManager
	viewDateManager viewDateManager
	oCounterManager oCounterManager
	omgCounterMgr   omgCounterManager
}

func NewGameStore() *GameStore {
	return &GameStore{
		tableMgr:        gameTableMgr,
		oDateManager:    oDateManager{gamesOTableMgr},
		omgDateManager:  omgDateManager{gamesOMGTableMgr},
		viewDateManager: viewDateManager{tableMgr: gamesViewTableMgr},
		oCounterManager: oCounterManager{tableMgr: gameTableMgr},
		omgCounterMgr:   omgCounterManager{tableMgr: gameTableMgr},
	}
}

func (qb *GameStore) table() exp.IdentifierExpression {
	return qb.tableMgr.table
}

func (qb *GameStore) selectDataset() *goqu.SelectDataset {
	return dialect.From(qb.table()).Select(qb.table().All())
}

func (qb *GameStore) Create(ctx context.Context, game *models.Game) error {
	var row gameRow
	row.fromGame(*game)

	id, err := qb.tableMgr.insertID(ctx, row)
	if err != nil {
		return err
	}

	if game.URLs.Loaded() {
		const startPos = 0
		if err := gamesURLsTableMgr.insertJoins(ctx, id, startPos, game.URLs.List()); err != nil {
			return err
		}
	}
	if game.TagIDs.Loaded() {
		if err := gamesTagsTableMgr.insertJoins(ctx, id, game.TagIDs.List()); err != nil {
			return err
		}
	}

	created, err := qb.find(ctx, id)
	if err != nil {
		return fmt.Errorf("finding game after create: %w", err)
	}

	*game = *created
	return nil
}

func (qb *GameStore) Update(ctx context.Context, game *models.Game) error {
	var row gameRow
	row.fromGame(*game)
	if err := qb.tableMgr.updateByID(ctx, game.ID, row); err != nil {
		return err
	}

	if game.URLs.Loaded() {
		if err := gamesURLsTableMgr.replaceJoins(ctx, game.ID, game.URLs.List()); err != nil {
			return err
		}
	}
	if game.TagIDs.Loaded() {
		if err := gamesTagsTableMgr.replaceJoins(ctx, game.ID, game.TagIDs.List()); err != nil {
			return err
		}
	}

	return nil
}

func (qb *GameStore) UpdatePartial(ctx context.Context, id int, partial models.GamePartial) (*models.Game, error) {
	record := gameRowRecord{
		updateRecord: updateRecord{
			Record: make(exp.Record),
		},
	}

	record.fromPartial(partial)
	if len(record.Record) > 0 {
		if err := qb.tableMgr.updateByID(ctx, id, record.Record); err != nil {
			return nil, err
		}
	}

	if partial.URLs != nil {
		if err := gamesURLsTableMgr.modifyJoins(ctx, id, partial.URLs.Values, partial.URLs.Mode); err != nil {
			return nil, err
		}
	}
	if partial.TagIDs != nil {
		if err := gamesTagsTableMgr.modifyJoins(ctx, id, partial.TagIDs.IDs, partial.TagIDs.Mode); err != nil {
			return nil, err
		}
	}

	return qb.find(ctx, id)
}

func (qb *GameStore) Destroy(ctx context.Context, id int) error {
	return qb.tableMgr.destroyExisting(ctx, []int{id})
}

func (qb *GameStore) Find(ctx context.Context, id int) (*models.Game, error) {
	ret, err := qb.find(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return ret, err
}

func (qb *GameStore) find(ctx context.Context, id int) (*models.Game, error) {
	q := qb.selectDataset().Prepared(true).Where(qb.tableMgr.byID(id))
	ret, err := qb.get(ctx, q)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (qb *GameStore) FindMany(ctx context.Context, ids []int) ([]*models.Game, error) {
	games := make([]*models.Game, len(ids))
	if err := batchExec(ids, defaultBatchSize, func(batch []int) error {
		q := qb.selectDataset().Prepared(true).Where(qb.table().Col(idColumn).In(batch))
		unsorted, err := qb.getMany(ctx, q)
		if err != nil {
			return err
		}

		for _, g := range unsorted {
			i := slices.Index(ids, g.ID)
			if i >= 0 && i < len(games) {
				games[i] = g
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}

	for i := range games {
		if games[i] == nil {
			return nil, fmt.Errorf("game with id %d not found", ids[i])
		}
	}

	return games, nil
}

func (qb *GameStore) All(ctx context.Context) ([]*models.Game, error) {
	q := qb.selectDataset().Prepared(true)
	return qb.getMany(ctx, q)
}

func (qb *GameStore) get(ctx context.Context, q *goqu.SelectDataset) (*models.Game, error) {
	ret, err := qb.getMany(ctx, q)
	if err != nil {
		return nil, err
	}

	if len(ret) == 0 {
		return nil, sql.ErrNoRows
	}

	return ret[0], nil
}

func (qb *GameStore) getMany(ctx context.Context, q *goqu.SelectDataset) ([]*models.Game, error) {
	const single = false
	var ret []*models.Game
	if err := queryFunc(ctx, q, single, func(r *sqlx.Rows) error {
		var row gameRow
		if err := r.StructScan(&row); err != nil {
			return err
		}
		ret = append(ret, row.resolve())
		return nil
	}); err != nil {
		return nil, err
	}
	return ret, nil
}

func (qb *GameStore) GetTagIDs(ctx context.Context, id int) ([]int, error) {
	return gamesTagsTableMgr.get(ctx, id)
}

func (qb *GameStore) GetURLs(ctx context.Context, id int) ([]string, error) {
	return gamesURLsTableMgr.get(ctx, id)
}

func (qb *GameStore) Query(ctx context.Context, gameFilter *models.GameFilterType, findFilter *models.FindFilterType) ([]*models.Game, int, error) {
	query, err := qb.makeQuery(ctx, gameFilter, findFilter)
	if err != nil {
		return nil, 0, err
	}

	ids, total, err := query.executeFind(ctx)
	if err != nil {
		return nil, 0, err
	}

	games, err := qb.FindMany(ctx, ids)
	if err != nil {
		return nil, 0, err
	}

	return games, total, nil
}

func (qb *GameStore) QueryCount(ctx context.Context, gameFilter *models.GameFilterType, findFilter *models.FindFilterType) (int, error) {
	query, err := qb.makeQuery(ctx, gameFilter, findFilter)
	if err != nil {
		return 0, err
	}

	return query.executeCount(ctx)
}

var gameSortOptions = sortOptions{
	"created_at",
	"date",
	"id",
	"o_counter",
	"omg_counter",
	"play_count",
	"rating",
	"tag_count",
	"title",
	"updated_at",
}

func (qb *GameStore) makeQuery(ctx context.Context, gameFilter *models.GameFilterType, findFilter *models.FindFilterType) (*queryBuilder, error) {
	query := &queryBuilder{
		repository: &gameRepository.repository,
		columns:    []string{"games.id"},
		from:       gameTable,
	}

	handler := &gameFilterHandler{gameFilter: gameFilter}
	if err := query.addFilter(filterBuilderFromHandler(ctx, handler.criterionHandler())); err != nil {
		return nil, err
	}

	if findFilter != nil && findFilter.Q != nil && *findFilter.Q != "" {
		query.parseQueryString([]string{"games.title", "games.details"}, *findFilter.Q)
	}

	if err := qb.setGameSortAndPagination(query, findFilter); err != nil {
		return nil, err
	}

	return query, nil
}

func (qb *GameStore) setGameSortAndPagination(query *queryBuilder, findFilter *models.FindFilterType) error {
	sort := "created_at"
	direction := "DESC"

	if findFilter != nil && findFilter.Sort != nil && *findFilter.Sort != "" {
		sort = findFilter.GetSort(sort)
		direction = findFilter.GetDirection()
	}

	if err := gameSortOptions.validateSort(sort); err != nil {
		return err
	}

	switch sort {
	case "tag_count":
		query.sortAndPagination = getCountSort(gameTable, gamesTagsTable, gameIDColumn, direction)
	case "play_count":
		query.sortAndPagination = getCountSort(gameTable, gamesViewDatesTable, gameIDColumn, direction)
	default:
		query.sortAndPagination = fmt.Sprintf(" ORDER BY games.%s %s", sort, getSortDirection(direction))
	}

	perPage := 25
	page := 1
	if findFilter != nil {
		if findFilter.IsGetAll() {
			return nil
		}
		perPage = findFilter.GetPageSize()
		page = findFilter.GetPage()
	}

	offset := (page - 1) * perPage
	query.sortAndPagination += fmt.Sprintf(" LIMIT %d OFFSET %d", perPage, offset)
	return nil
}

func (qb *GameStore) IncrementOCounter(ctx context.Context, id int) (int, error) {
	return qb.oCounterManager.IncrementOCounter(ctx, id)
}

func (qb *GameStore) DecrementOCounter(ctx context.Context, id int) (int, error) {
	return qb.oCounterManager.DecrementOCounter(ctx, id)
}

func (qb *GameStore) ResetOCounter(ctx context.Context, id int) (int, error) {
	return qb.oCounterManager.ResetOCounter(ctx, id)
}

func (qb *GameStore) IncrementOMGCounter(ctx context.Context, id int) (int, error) {
	return qb.omgCounterMgr.IncrementOMGCounter(ctx, id)
}

func (qb *GameStore) DecrementOMGCounter(ctx context.Context, id int) (int, error) {
	return qb.omgCounterMgr.DecrementOMGCounter(ctx, id)
}

func (qb *GameStore) ResetOMGCounter(ctx context.Context, id int) (int, error) {
	return qb.omgCounterMgr.ResetOMGCounter(ctx, id)
}

func (qb *GameStore) AddO(ctx context.Context, id int, dates []time.Time) ([]time.Time, error) {
	return qb.oDateManager.AddO(ctx, id, dates)
}

func (qb *GameStore) DeleteO(ctx context.Context, id int, dates []time.Time) ([]time.Time, error) {
	return qb.oDateManager.DeleteO(ctx, id, dates)
}

func (qb *GameStore) ResetO(ctx context.Context, id int) (int, error) {
	return qb.oDateManager.ResetO(ctx, id)
}

func (qb *GameStore) AddOMG(ctx context.Context, id int, dates []time.Time) ([]time.Time, error) {
	return qb.omgDateManager.AddOMG(ctx, id, dates)
}

func (qb *GameStore) DeleteOMG(ctx context.Context, id int, dates []time.Time) ([]time.Time, error) {
	return qb.omgDateManager.DeleteOMG(ctx, id, dates)
}

func (qb *GameStore) ResetOMG(ctx context.Context, id int) (int, error) {
	return qb.omgDateManager.ResetOMG(ctx, id)
}

func (qb *GameStore) CountViews(ctx context.Context, id int) (int, error) {
	return qb.viewDateManager.CountViews(ctx, id)
}

func (qb *GameStore) AddViews(ctx context.Context, id int, dates []time.Time) ([]time.Time, error) {
	return qb.viewDateManager.AddViews(ctx, id, dates)
}

func (qb *GameStore) DeleteViews(ctx context.Context, id int, dates []time.Time) ([]time.Time, error) {
	return qb.viewDateManager.DeleteViews(ctx, id, dates)
}

func (qb *GameStore) DeleteAllViews(ctx context.Context, id int) (int, error) {
	return qb.viewDateManager.DeleteAllViews(ctx, id)
}

func (qb *GameStore) GetODates(ctx context.Context, id int) ([]time.Time, error) {
	return qb.oDateManager.GetODates(ctx, id)
}

func (qb *GameStore) GetOMGDates(ctx context.Context, id int) ([]time.Time, error) {
	return qb.omgDateManager.GetOMGDates(ctx, id)
}

func (qb *GameStore) GetViewDates(ctx context.Context, id int) ([]time.Time, error) {
	return qb.viewDateManager.GetViewDates(ctx, id)
}

func (qb *GameStore) GetManyOCount(ctx context.Context, ids []int) ([]int, error) {
	return qb.oDateManager.GetManyOCount(ctx, ids)
}

func (qb *GameStore) GetManyODates(ctx context.Context, ids []int) ([][]time.Time, error) {
	return qb.oDateManager.GetManyODates(ctx, ids)
}

func (qb *GameStore) GetManyOMGCount(ctx context.Context, ids []int) ([]int, error) {
	return qb.omgDateManager.GetManyOMGCount(ctx, ids)
}

func (qb *GameStore) GetManyOMGDates(ctx context.Context, ids []int) ([][]time.Time, error) {
	return qb.omgDateManager.GetManyOMGDates(ctx, ids)
}

func (qb *GameStore) GetManyViewCount(ctx context.Context, ids []int) ([]int, error) {
	return qb.viewDateManager.GetManyViewCount(ctx, ids)
}

func (qb *GameStore) GetManyViewDates(ctx context.Context, ids []int) ([][]time.Time, error) {
	return qb.viewDateManager.GetManyViewDates(ctx, ids)
}

func (qb *GameStore) GetImage(ctx context.Context, gameID int) ([]byte, error) {
	table := qb.table()
	q := dialect.From(table).Select(table.Col("image")).Where(table.Col("id").Eq(gameID))

	var image []byte
	const single = true
	if err := queryFunc(ctx, q, single, func(r *sqlx.Rows) error {
		return r.Scan(&image)
	}); err != nil {
		return nil, fmt.Errorf("querying game image: %w", err)
	}

	return image, nil
}

func (qb *GameStore) HasImage(ctx context.Context, gameID int) (bool, error) {
	image, err := qb.GetImage(ctx, gameID)
	if err != nil {
		return false, err
	}
	return len(image) > 0, nil
}
