package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jullss/banners-rotation/internal/bandit"
	"github.com/jullss/banners-rotation/internal/domain"
)

type Storage struct {
	db      *sql.DB
	chooser bandit.Chooser
}

func New(dsn string, chooser bandit.Chooser) (*Storage, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	if err := db.PingContext(context.Background()); err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}
	return &Storage{db: db, chooser: chooser}, nil
}

func (s *Storage) Close() error {
	return s.db.Close()
}

func (s *Storage) DB() *sql.DB {
	return s.db
}

func (s *Storage) AddBannerToSlot(ctx context.Context, slotID, bannerID int64) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO slot_banners (slot_id, banner_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		slotID, bannerID,
	)
	if err != nil {
		return fmt.Errorf("add banner to slot: %w", err)
	}
	return nil
}

func (s *Storage) RemoveBannerFromSlot(ctx context.Context, slotID, bannerID int64) error {
	_, err := s.db.ExecContext(ctx,
		`DELETE FROM slot_banners WHERE slot_id = $1 AND banner_id = $2`,
		slotID, bannerID,
	)
	if err != nil {
		return fmt.Errorf("remove banner from slot: %w", err)
	}
	return nil
}

func (s *Storage) RecordClick(ctx context.Context, slotID, bannerID, groupID int64) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO stats (slot_id, banner_id, group_id, clicks, shows)
		VALUES ($1, $2, $3, 1, 0)
		ON CONFLICT (slot_id, banner_id, group_id)
		DO UPDATE SET clicks = stats.clicks + 1`,
		slotID, bannerID, groupID,
	)
	if err != nil {
		return fmt.Errorf("record click: %w", err)
	}
	return nil
}

func (s *Storage) ChooseBanner(ctx context.Context, slotID, groupID int64) (*domain.Banner, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT b.id, b.description,
		       COALESCE(s.shows, 0)  AS shows,
		       COALESCE(s.clicks, 0) AS clicks
		FROM slot_banners sb
		JOIN banners b ON b.id = sb.banner_id
		LEFT JOIN stats s
		       ON s.slot_id   = sb.slot_id
		      AND s.banner_id = sb.banner_id
		      AND s.group_id  = $2
		WHERE sb.slot_id = $1`,
		slotID, groupID,
	)
	if err != nil {
		return nil, fmt.Errorf("query banners: %w", err)
	}
	defer rows.Close()

	var stats []domain.Stat
	banners := make(map[int64]*domain.Banner)

	for rows.Next() {
		var b domain.Banner
		var st domain.Stat
		if err := rows.Scan(&b.ID, &b.Description, &st.Shows, &st.Clicks); err != nil {
			return nil, fmt.Errorf("scan banner: %w", err)
		}
		st.BannerID = b.ID
		st.SlotID = slotID
		st.GroupID = groupID
		stats = append(stats, st)
		banners[b.ID] = &domain.Banner{ID: b.ID, Description: b.Description}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	if len(stats) == 0 {
		return nil, errors.New("no banners in slot")
	}

	chosenID := s.chooser.Choose(stats)

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO stats (slot_id, banner_id, group_id, shows, clicks)
		VALUES ($1, $2, $3, 1, 0)
		ON CONFLICT (slot_id, banner_id, group_id)
		DO UPDATE SET shows = stats.shows + 1`,
		slotID, chosenID, groupID,
	)
	if err != nil {
		return nil, fmt.Errorf("record show: %w", err)
	}

	return banners[chosenID], nil
}
