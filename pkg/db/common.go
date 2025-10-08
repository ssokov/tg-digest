package db

import (
	"context"
	"errors"

	"github.com/go-pg/pg/v10"
	"github.com/go-pg/pg/v10/orm"
)

type CommonRepo struct {
	db      orm.DB
	filters map[string][]Filter
	sort    map[string][]SortField
	join    map[string][]string
}

// NewCommonRepo returns new repository
func NewCommonRepo(db orm.DB) CommonRepo {
	return CommonRepo{
		db:      db,
		filters: map[string][]Filter{},
		sort: map[string][]SortField{
			Tables.MessageReaction.Name: {{Column: Columns.MessageReaction.CreatedAt, Direction: SortDesc}},
		},
		join: map[string][]string{
			Tables.MessageReaction.Name: {TableColumns},
		},
	}
}

// WithTransaction is a function that wraps CommonRepo with pg.Tx transaction.
func (cr CommonRepo) WithTransaction(tx *pg.Tx) CommonRepo {
	cr.db = tx
	return cr
}

// WithEnabledOnly is a function that adds "statusId"=1 as base filter.
func (cr CommonRepo) WithEnabledOnly() CommonRepo {
	f := make(map[string][]Filter, len(cr.filters))
	for i := range cr.filters {
		f[i] = make([]Filter, len(cr.filters[i]))
		copy(f[i], cr.filters[i])
		f[i] = append(f[i], StatusEnabledFilter)
	}
	cr.filters = f

	return cr
}

/*** MessageReaction ***/

// FullMessageReaction returns full joins with all columns
func (cr CommonRepo) FullMessageReaction() OpFunc {
	return WithColumns(cr.join[Tables.MessageReaction.Name]...)
}

// DefaultMessageReactionSort returns default sort.
func (cr CommonRepo) DefaultMessageReactionSort() OpFunc {
	return WithSort(cr.sort[Tables.MessageReaction.Name]...)
}

// MessageReactionByID is a function that returns MessageReaction by ID(s) or nil.
func (cr CommonRepo) MessageReactionByID(ctx context.Context, messageID int, chatID int64, ops ...OpFunc) (*MessageReaction, error) {
	return cr.OneMessageReaction(ctx, &MessageReactionSearch{MessageID: &messageID, ChatID: &chatID}, ops...)
}

// OneMessageReaction is a function that returns one MessageReaction by filters. It could return pg.ErrMultiRows.
func (cr CommonRepo) OneMessageReaction(ctx context.Context, search *MessageReactionSearch, ops ...OpFunc) (*MessageReaction, error) {
	obj := &MessageReaction{}
	err := buildQuery(ctx, cr.db, obj, search, cr.filters[Tables.MessageReaction.Name], PagerTwo, ops...).Select()

	if errors.Is(err, pg.ErrMultiRows) {
		return nil, err
	} else if errors.Is(err, pg.ErrNoRows) {
		return nil, nil
	}

	return obj, err
}

// MessageReactionsByFilters returns MessageReaction list.
func (cr CommonRepo) MessageReactionsByFilters(ctx context.Context, search *MessageReactionSearch, pager Pager, ops ...OpFunc) (messageReactions []MessageReaction, err error) {
	err = buildQuery(ctx, cr.db, &messageReactions, search, cr.filters[Tables.MessageReaction.Name], pager, ops...).Select()
	return
}

// CountMessageReactions returns count
func (cr CommonRepo) CountMessageReactions(ctx context.Context, search *MessageReactionSearch, ops ...OpFunc) (int, error) {
	return buildQuery(ctx, cr.db, &MessageReaction{}, search, cr.filters[Tables.MessageReaction.Name], PagerOne, ops...).Count()
}

// AddMessageReaction adds MessageReaction to DB.
func (cr CommonRepo) AddMessageReaction(ctx context.Context, messageReaction *MessageReaction, ops ...OpFunc) (*MessageReaction, error) {
	q := cr.db.ModelContext(ctx, messageReaction)
	if len(ops) == 0 {
		q = q.ExcludeColumn(Columns.MessageReaction.CreatedAt)
	}
	applyOps(q, ops...)
	_, err := q.Insert()

	return messageReaction, err
}

// UpdateMessageReaction updates MessageReaction in DB.
func (cr CommonRepo) UpdateMessageReaction(ctx context.Context, messageReaction *MessageReaction, ops ...OpFunc) (bool, error) {
	q := cr.db.ModelContext(ctx, messageReaction).WherePK()
	if len(ops) == 0 {
		q = q.ExcludeColumn(Columns.MessageReaction.MessageID, Columns.MessageReaction.ChatID, Columns.MessageReaction.CreatedAt)
	}
	applyOps(q, ops...)
	res, err := q.Update()
	if err != nil {
		return false, err
	}

	return res.RowsAffected() > 0, err
}

// DeleteMessageReaction deletes MessageReaction from DB.
func (cr CommonRepo) DeleteMessageReaction(ctx context.Context, messageID int, chatID int64) (deleted bool, err error) {
	messageReaction := &MessageReaction{MessageID: messageID, ChatID: chatID}

	res, err := cr.db.ModelContext(ctx, messageReaction).WherePK().Delete()
	if err != nil {
		return false, err
	}

	return res.RowsAffected() > 0, err
}
