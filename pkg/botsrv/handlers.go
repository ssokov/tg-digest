package botsrv

import (
	"botsrv/pkg/db"
	"botsrv/pkg/embedlog"
	"context"
	"fmt"

	"github.com/go-pg/pg/v10"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

const (
	startCommand  = "/start"
	digestCommand = "/digest"
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
}

func (bm *BotManager) DefaultHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	bm.Printf("%+v", update)
	if update.MessageReaction != nil {
		println("пришел реакт")
		if err := bm.dbo.RunInTransaction(ctx, func(tx *pg.Tx) error {
			crTx := bm.cr.WithTransaction(tx)
			mr, err := crTx.OneMessageReaction(ctx, &db.MessageReactionSearch{
				ID:     &update.MessageReaction.MessageID,
				ChatID: &update.MessageReaction.Chat.ID,
			})
			if err != nil {
				return err
			}
			if mr == nil {
				_, err = crTx.AddMessageReaction(ctx, &db.MessageReaction{
					ID:     update.MessageReaction.MessageID,
					ChatID: update.MessageReaction.Chat.ID,
					//  Сделал, чтобы ставилось сначала текущее количество реакий
					//  т.к. может сразу прийти больше 1 реакции
					ReactionsCount: pointer(len(update.MessageReaction.NewReaction)),
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
	//if update.Message == nil {
	//	return
	//}
	//_, err := b.SendMessage(ctx, &bot.SendMessageParams{
	//	ChatID: update.Message.Chat.ID,
	//	Text:   "test",
	//})
	//if err != nil {
	//	bm.Errorf("%v", err)
	//	return
	//}
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
	reactions, err := bm.cr.MessageReactionsByFilters(ctx, &db.MessageReactionSearch{
		// фильтр по чату
		ChatID: &update.Message.Chat.ID,
	}, db.Pager{PageSize: 10},
		db.WithSort(db.NewSortField(db.Columns.MessageReaction.ReactionsCount, true)))
	if err != nil {
		bm.Errorf("%v", err)
		return
	}
	res := "Топ реакции:"
	for _, reaction := range reactions {
		link := fmt.Sprintf("https://t.me/%s/%d", update.Message.Chat.Username, reaction.ID)
		res += fmt.Sprintf("\nCount: %d Link: %s", *reaction.ReactionsCount, link)
	}
	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   res,
	})
	if err != nil {
		bm.Errorf("%v", err)
		return
	}
}

func pointer[T any](in T) *T { return &in }
