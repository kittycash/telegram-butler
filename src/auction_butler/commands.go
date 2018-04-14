package auction_butler

import (
	"time"
	"fmt"
	"strings"
	"github.com/bcampbell/fuzzytime"
	"errors"
)

type Command struct {
	Admin       bool
	Command     string
	Handlerfunc CommandHandler
}

type Commands []Command

func (bot *Bot) setCommandHandlers() {
	for _, command := range commands {
		bot.SetCommandHandler(command.Admin, command.Command, command.Handlerfunc)
	}
}

func (bot *Bot) AddPrivateMessageHandler(handler MessageHandler) {
	bot.privateMessageHandlers = append(bot.privateMessageHandlers, handler)
}

func (bot *Bot) AddGroupMessageHandler(handler MessageHandler) {
	bot.groupMessageHandlers = append(bot.groupMessageHandlers, handler)
}

// Handler for help command
func (bot *Bot) handleCommandHelp(ctx *Context, command, args string) error {
	// Indentation messes up how the text is shown in chat.
	if ctx.User.Admin {
		return bot.Reply(ctx, `
/start
/help - this text
/setauctioninfo [end_time] [auction_info](optional) - set details for current auction
/getauctioninfo - returns info of current auction
`)
	}

	return bot.Reply(ctx, `
/start
/help - this text
/getauctioninfo - returns info of current auction`)
}

func (bot *Bot) handleSetAuctionInfo(ctx *Context, command, args string) error {
	auction := bot.currentAuction
	if auction != nil {
		return bot.Reply(ctx, fmt.Sprintf("An auction is already scheduled to end at %s", auction.EndTime.Time.UTC().String()))
	}

	end, err := parseStartAuctionArgs(args)
	if err != nil {
		return fmt.Errorf("could not parse the date: %v", err)
	}

	bot.runningCountDown = false
	auction.EndTime = NewNullTime(end)
	err = bot.db.PutAuction(auction)
	if err != nil {
		return bot.Reply(ctx, fmt.Sprintf("failed to set an auction: %v", err))
	} else {
		bot.Reschedule()
		return bot.Reply(ctx, fmt.Sprintf("Auction scheduled to end at: %s", end.UTC().String()))
	}
}

func (bot *Bot) handleEditAuctionInfo(ctx *Context, command, args string) error {
	auction := bot.currentAuction
	if auction == nil {
		return bot.Reply(ctx, fmt.Sprintf("No auction is scheduled"))
	}

	end, err := parseStartAuctionArgs(args)
	if err != nil {
		return fmt.Errorf("could not parse the date: %v", err)
	}

	// update the auction info
	auction.EndTime = NewNullTime(end)
	err = bot.db.PutAuction(auction)
	if err != nil {
		return bot.Reply(ctx, fmt.Sprintf("unable to update auction: %v", err))
	}

	// reschedule the bot
	bot.Reschedule()

	return bot.Reply(ctx, "Auction updated!")
}


//func (bot *Bot) handleAnnounce(ctx, command, args string)

func (bot *Bot) handleGetAuctionInfo(ctx *Context, command, args string) error {
    auction := bot.currentAuction
	if auction == nil {
		return errors.New("No auction found")
	}
    return bot.Reply(ctx, fmt.Sprintf(`Auction End Time: %s`,  auction.EndTime.Time.UTC().String()))
}

func parseStartAuctionArgs(args string) (end time.Time, err error) {
	words := strings.Fields(args)
	if len(words) == 0 {
		err = fmt.Errorf("insufficient arguments")
		return
	}

	timestr := strings.Join(words, " ")
	ft, _, err := fuzzytime.Extract(timestr)
	if ft.Empty() {
		err = fmt.Errorf("unsupported datetime format: %v", timestr)
		return
	}

	var hour, minute, second int
	var loc *time.Location
	if ft.Time.HasHour() {
		hour = ft.Time.Hour()
	}
	if ft.Time.HasMinute() {
		minute = ft.Time.Minute()
	}
	if ft.Time.HasSecond() {
		second = ft.Time.Second()
	}
	if ft.Time.HasTZOffset() {
		loc = time.FixedZone("", ft.Time.TZOffset())
	} else {
		loc = time.UTC
	}

	if ft.HasFullDate() {
		end = time.Date(
			ft.Date.Year(),
			time.Month(ft.Date.Month()),
			ft.Date.Day(),
			hour, minute, second, 0,
			loc,
		)
	} else {
		year, month, day := time.Now().In(loc).Date()
		end = time.Date(
			year, month, day,
			hour, minute, second, 0,
			loc,
		)
		if end.Before(time.Now()) {
			end = end.AddDate(0, 0, 1)
		}
	}

	if end.Before(time.Now()) {
		err = fmt.Errorf("%s is in the past", end.String())
		return
	}

	return
}

func (bot *Bot) SetCommandHandler(admin bool, command string, handler CommandHandler) {
	if admin {
		bot.adminCommandHandlers[command] = handler
	} else {
		bot.commandHandlers[command] = handler
	}
}

var commands = Commands{
	Command{
		false,
		"help",
		(*Bot).handleCommandHelp,
	},
	Command{
		false,
		"getauctioninfo",
		(*Bot).handleGetAuctionInfo,
	},
	Command{
		true,
		"setauctioninfo",
		(*Bot).handleSetAuctionInfo,
	},
	Command{
		 true,
		"editauctioninfo",
		(*Bot).handleEditAuctionInfo,
	},
	//Command{
	//	true,
	//		"announce",
	//	(*Bot).handleAnnounce,
	//},
}
