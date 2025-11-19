package sqlite

import (
	"context"
	"time"

	"github.com/stashapp/stash/pkg/models"
)

type viewDateManager struct {
	tableMgr *viewHistoryTable
}

func (qb *viewDateManager) GetViewDates(ctx context.Context, id int) ([]time.Time, error) {
	return qb.tableMgr.getDates(ctx, id)
}

func (qb *viewDateManager) GetManyViewDates(ctx context.Context, ids []int) ([][]time.Time, error) {
	return qb.tableMgr.getManyDates(ctx, ids)
}

func (qb *viewDateManager) CountViews(ctx context.Context, id int) (int, error) {
	return qb.tableMgr.getCount(ctx, id)
}

func (qb *viewDateManager) GetManyViewCount(ctx context.Context, ids []int) ([]int, error) {
	return qb.tableMgr.getManyCount(ctx, ids)
}

func (qb *viewDateManager) CountAllViews(ctx context.Context) (int, error) {
	return qb.tableMgr.getAllCount(ctx)
}

func (qb *viewDateManager) CountUniqueViews(ctx context.Context) (int, error) {
	return qb.tableMgr.getUniqueCount(ctx)
}

func (qb *viewDateManager) GetAggregatedViewHistory(ctx context.Context, page, perPage int) ([]models.AggregatedView, error) {
	// Get all aggregated view history and o_dates in one query using JOIN
	query := `
		SELECT
			gv.scene_id,
			gv.latest_view_date as view_date,
			gv.view_count,
			(
				SELECT sod.o_date
				FROM scenes_o_dates sod
				WHERE sod.scene_id = gv.scene_id
				AND sod.o_date > gv.earliest_view_date
				ORDER BY sod.o_date ASC
				LIMIT 1
			) as o_date,
			(
				SELECT somgd.omg_date
				FROM scenes_omg_dates somgd
				WHERE somgd.scene_id = gv.scene_id
				AND somgd.omg_date > gv.earliest_view_date
				ORDER BY somgd.omg_date ASC
				LIMIT 1
			) as omg_date
		FROM (
			SELECT
				svd.scene_id,
				COUNT(*) as view_count,
				MIN(svd.view_date) as earliest_view_date,
				MAX(svd.view_date) as latest_view_date
			FROM scenes_view_dates svd
			GROUP BY svd.scene_id, DATE(svd.view_date)
			ORDER BY MAX(svd.view_date) DESC
		) gv
	`

	rows, err := dbWrapper.QueryxContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var allResults []models.AggregatedView
	resultMap := make(map[int]*models.AggregatedView)

	for rows.Next() {
		var sceneID int
		var viewDateStr string
		var viewCount int
		var oDateStr *string
		var omgDateStr *string

		err := rows.Scan(&sceneID, &viewDateStr, &viewCount, &oDateStr, &omgDateStr)
		if err != nil {
			return nil, err
		}

		// Parse the view date
		viewDate, err := time.Parse(time.RFC3339, viewDateStr)
		if err != nil {
			return nil, err
		}

		// Check if we already have this scene
		if av, exists := resultMap[sceneID]; exists {
			// If we have o_date, update it (take the first one)
			if oDateStr != nil && av.ODate == nil {
				oDate, err := time.Parse(time.RFC3339, *oDateStr)
				if err != nil {
					return nil, err
				}
				av.ODate = &oDate
			}
			// If we have omg_date, update it (take the first one)
			if omgDateStr != nil && av.OmgDate == nil {
				omgDate, err := time.Parse(time.RFC3339, *omgDateStr)
				if err != nil {
					return nil, err
				}
				av.OmgDate = &omgDate
			}
		} else {
			// Create new entry
			av := &models.AggregatedView{
				SceneID:   sceneID,
				ViewDate:  viewDate,
				ViewCount: viewCount,
			}

			if oDateStr != nil {
				oDate, err := time.Parse(time.RFC3339, *oDateStr)
				if err != nil {
					return nil, err
				}
				av.ODate = &oDate
			}

			if omgDateStr != nil {
				omgDate, err := time.Parse(time.RFC3339, *omgDateStr)
				if err != nil {
					return nil, err
				}
				av.OmgDate = &omgDate
			}

			resultMap[sceneID] = av
			allResults = append(allResults, *av)
		}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Apply pagination in Go code
	offset := (page - 1) * perPage
	if offset >= len(allResults) {
		return []models.AggregatedView{}, nil
	}

	end := offset + perPage
	if end > len(allResults) {
		end = len(allResults)
	}

	return allResults[offset:end], nil
}

func (qb *viewDateManager) GetAggregatedViewHistoryCount(ctx context.Context) (int, error) {
	query := `SELECT COUNT(*) FROM scenes_view_dates`

	var count int
	err := dbWrapper.Get(ctx, &count, query)
	return count, err
}

func (qb *viewDateManager) LastView(ctx context.Context, id int) (*time.Time, error) {
	return qb.tableMgr.getLastDate(ctx, id)
}

func (qb *viewDateManager) GetManyLastViewed(ctx context.Context, ids []int) ([]*time.Time, error) {
	return qb.tableMgr.getManyLastDate(ctx, ids)

}

func (qb *viewDateManager) AddViews(ctx context.Context, id int, dates []time.Time) ([]time.Time, error) {
	return qb.tableMgr.addDates(ctx, id, dates)
}

func (qb *viewDateManager) DeleteViews(ctx context.Context, id int, dates []time.Time) ([]time.Time, error) {
	return qb.tableMgr.deleteDates(ctx, id, dates)
}

func (qb *viewDateManager) DeleteAllViews(ctx context.Context, id int) (int, error) {
	return qb.tableMgr.deleteAllDates(ctx, id)
}

type oDateManager struct {
	tableMgr *viewHistoryTable
}

func (qb *oDateManager) GetODates(ctx context.Context, id int) ([]time.Time, error) {
	return qb.tableMgr.getDates(ctx, id)
}

func (qb *oDateManager) GetManyODates(ctx context.Context, ids []int) ([][]time.Time, error) {
	return qb.tableMgr.getManyDates(ctx, ids)
}

func (qb *oDateManager) GetOCount(ctx context.Context, id int) (int, error) {
	return qb.tableMgr.getCount(ctx, id)
}

func (qb *oDateManager) GetManyOCount(ctx context.Context, ids []int) ([]int, error) {
	return qb.tableMgr.getManyCount(ctx, ids)
}

func (qb *oDateManager) GetAllOCount(ctx context.Context) (int, error) {
	return qb.tableMgr.getAllCount(ctx)
}

func (qb *oDateManager) GetUniqueOCount(ctx context.Context) (int, error) {
	return qb.tableMgr.getUniqueCount(ctx)
}

func (qb *oDateManager) AddO(ctx context.Context, id int, dates []time.Time) ([]time.Time, error) {
	return qb.tableMgr.addDates(ctx, id, dates)
}

func (qb *oDateManager) DeleteO(ctx context.Context, id int, dates []time.Time) ([]time.Time, error) {
	return qb.tableMgr.deleteDates(ctx, id, dates)
}

func (qb *oDateManager) ResetO(ctx context.Context, id int) (int, error) {
	return qb.tableMgr.deleteAllDates(ctx, id)
}

func (qb *oDateManager) GetODatesInRange(ctx context.Context, start, end time.Time) ([]time.Time, error) {
	return qb.tableMgr.getDatesInRange(ctx, start, end)
}

type omgDateManager struct {
	tableMgr *viewHistoryTable
}

func (qb *omgDateManager) GetOMGDates(ctx context.Context, id int) ([]time.Time, error) {
	return qb.tableMgr.getDates(ctx, id)
}

func (qb *omgDateManager) GetManyOMGDates(ctx context.Context, ids []int) ([][]time.Time, error) {
	return qb.tableMgr.getManyDates(ctx, ids)
}

func (qb *omgDateManager) GetOMGCount(ctx context.Context, id int) (int, error) {
	return qb.tableMgr.getCount(ctx, id)
}

func (qb *omgDateManager) GetManyOMGCount(ctx context.Context, ids []int) ([]int, error) {
	return qb.tableMgr.getManyCount(ctx, ids)
}

func (qb *omgDateManager) GetAllOMGCount(ctx context.Context) (int, error) {
	return qb.tableMgr.getAllCount(ctx)
}

func (qb *omgDateManager) GetUniqueOMGCount(ctx context.Context) (int, error) {
	return qb.tableMgr.getUniqueCount(ctx)
}

func (qb *omgDateManager) AddOMG(ctx context.Context, id int, dates []time.Time) ([]time.Time, error) {
	return qb.tableMgr.addDates(ctx, id, dates)
}

func (qb *omgDateManager) DeleteOMG(ctx context.Context, id int, dates []time.Time) ([]time.Time, error) {
	return qb.tableMgr.deleteDates(ctx, id, dates)
}

func (qb *omgDateManager) ResetOMG(ctx context.Context, id int) (int, error) {
	return qb.tableMgr.deleteAllDates(ctx, id)
}

func (qb *omgDateManager) GetOMGDatesInRange(ctx context.Context, start, end time.Time) ([]time.Time, error) {
	return qb.tableMgr.getDatesInRange(ctx, start, end)
}
