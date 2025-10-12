package botsrv

import (
	"botsrv/pkg/db"
	"botsrv/pkg/embedlog"
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-pg/pg/v10"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

const (
	startCommand  = "/start"
	digestCommand = "/digest"

	patternDigestHour  = "digest:hour"
	patternDigestDay   = "digest:day"
	patternDigestWeek  = "digest:week"
	patternDigestMonth = "digest:month"
	patternDigestAll   = "digest:all"
	paternDigest       = "digest:"
)

type Config struct {
	Token string
}

type BotManager struct {
	embedlog.Logger
	dbo db.DB
	cr  db.CommonRepo
}

func NewBotManager(logger embedlog.Logger, dbo db.DB) *BotManager {
	return &BotManager{
		Logger: logger,
		dbo:    dbo,
		cr:     db.NewCommonRepo(dbo),
	}
}

func (bm *BotManager) RegisterBotHandlers(b *bot.Bot) {
	b.RegisterHandler(bot.HandlerTypeMessageText, startCommand, bot.MatchTypePrefix, bm.StartHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, digestCommand, bot.MatchTypePrefix, bm.DigestHandler)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, paternDigest, bot.MatchTypePrefix, bm.DigestCallbackHandler)
}

func (bm *BotManager) DefaultHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.MessageReaction != nil {
		if err := bm.dbo.RunInTransaction(ctx, func(tx *pg.Tx) error {
			crTx := bm.cr.WithTransaction(tx)
			mr, err := crTx.OneMessageReaction(ctx, &db.MessageReactionSearch{
				MessageID: &update.MessageReaction.MessageID,
				ChatID:    &update.MessageReaction.Chat.ID,
			})
			if err != nil {
				return err
			}
			if mr == nil {
				bm.Printf("Creating new message reaction for message ID: %d", update.MessageReaction.MessageID)
				_, err = crTx.AddMessageReaction(ctx, &db.MessageReaction{
					MessageID:      update.MessageReaction.MessageID,
					ChatID:         update.MessageReaction.Chat.ID,
					ReactionsCount: pointer(1),
				})
				if err != nil {
					return err
				}
			} else {
				*mr.ReactionsCount += len(update.MessageReaction.NewReaction) - len(update.MessageReaction.OldReaction)
				_, err = crTx.UpdateMessageReaction(ctx, mr)
				if err != nil {
					return err
				}
			}
			return nil
		}); err != nil {
			bm.Errorf("%v", err)
			return
		}
	}
}

func (bm *BotManager) StartHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}
	_, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   "start",
	})
	if err != nil {
		bm.Errorf("%v", err)
		return
	}
}

func (bm *BotManager) DigestHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}
	kb := &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{Text: "За час", CallbackData: patternDigestHour},
				{Text: "За день", CallbackData: patternDigestDay},
			},
			{
				{Text: "За неделю", CallbackData: patternDigestWeek},
				{Text: "За месяц", CallbackData: patternDigestMonth},
			},
			{
				{Text: "За всё время", CallbackData: patternDigestAll},
			},
		},
	}
	_, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:          update.Message.Chat.ID,
		Text:            "Выберите интервал для дайджеста:",
		ReplyMarkup:     kb,
		MessageThreadID: update.Message.MessageThreadID,
	})
	if err != nil {
		bm.Errorf("%v", err)
		return
	}
}

type ReactionsPeriod struct {
	Title  string
	Period time.Duration
}

var reactionPeriods = map[string]ReactionsPeriod{
	patternDigestHour: {
		Title:  "час",
		Period: 1 * time.Hour,
	},
	patternDigestDay: {
		Title:  "день",
		Period: 24 * time.Hour,
	},
	patternDigestWeek: {
		Title:  "неделю",
		Period: 24 * 7 * time.Hour,
	},
	patternDigestMonth: {
		Title:  "месяц",
		Period: 24 * 30 * time.Hour,
	},
	patternDigestAll: {
		Title: "всё время",
	},
}

func (bm *BotManager) DigestCallbackHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.CallbackQuery == nil || update.CallbackQuery.Data == "" {
		return
	}

	if !strings.HasPrefix(update.CallbackQuery.Data, "digest:") {
		return
	}
	periodPatern := update.CallbackQuery.Data
	bm.Printf("Processing digest callback with period: %s", periodPatern)
	chat := update.CallbackQuery.Message.Message.Chat

	chatID := chat.ID
	messageID := update.CallbackQuery.Message.Message.ID

	pageSize := 10

	now := time.Now()
	var period time.Time

	pattern, ok := reactionPeriods[periodPatern]
	if !ok {
		bm.Errorf("incorrect periodPatern")
		return
	}

	period = now.Add(-pattern.Period)
	if periodPatern == patternDigestAll {
		period = time.Unix(0, 0)
	}

	reactions, err := bm.cr.MessageReactionsByFilters(ctx, &db.MessageReactionSearch{
		ReactionsPeriod: &period,
		ChatID:          &chat.ID,
	}, db.Pager{PageSize: pageSize},
		db.WithSort(db.NewSortField(db.Columns.MessageReaction.ReactionsCount, true)))
	if err != nil {
		bm.Errorf("Failed to fetch message reactions: %v", err)
		return
	}

	bm.Printf("Retrieved %d reactions for chat %d", len(reactions), chat.ID)

	res := fmt.Sprintf("Топ реакции в чате за %s:", pattern.Title)
	for _, reaction := range reactions {
		var link string

		chatIDStr := strconv.FormatInt(-chat.ID-1000000000000, 10)
		switch update.CallbackQuery.Message.Message.Chat.Type {
		case models.ChatTypeGroup:
			link = fmt.Sprintf("https://t.me/%s/%d", chat.Username, reaction.MessageID)
		case models.ChatTypeSupergroup:
			link = fmt.Sprintf("https://t.me/c/%s/%d", chatIDStr, reaction.MessageID)
			if thread := update.CallbackQuery.Message.Message.MessageThreadID; thread != 0 {
				link += fmt.Sprintf("?thread=%d", thread)
			}
		}

		count := 0
		if reaction.ReactionsCount != nil {
			count = *reaction.ReactionsCount
		}
		res += fmt.Sprintf("\nРеакций: %d Ссылка: %s", count, link)
	}

	_, err = b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:    chatID,
		MessageID: messageID,
		Text:      res,
	})
	if err != nil {
		bm.Errorf("%v", err)
		return
	}
}

func pointer[T any](in T) *T { return &in }
