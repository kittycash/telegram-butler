package auction_butler

import (
	"fmt"
	"time"
)

type task int

const (
	nothing              task = iota
	endAuction
	reminderAnnouncement
	startCountDown
)

// Returns what to do next (start, stop or nothing) and when
func (bot *Bot) schedule() (task, time.Time) {
	if bot.runningCountDown {
		return nothing, time.Time{}
	}
	auction := bot.currentAuction
	if auction == nil {
		return nothing, time.Time{}
	}

	if auction.EndTime.Valid {
		bot.auctionEndTime = auction.EndTime.Time
		return endAuction, auction.EndTime.Time.Add(time.Second * -60)
	}

	return nothing, time.Time{}

}

// Returns a more detailed version than `schedule()`
// of what to do next (including announcements).
func (bot *Bot) subSchedule() (task, time.Time) {
	if bot.runningCountDown {
		return nothing, time.Now().Add(time.Second * 10)
	}
	tsk, future := bot.schedule()
	if tsk == nothing {
		return nothing, time.Now().Add(time.Second * 10)
	}

	// at what intervals to send the reminder for time left
	//TODO (therealssj): decrease reminder announce interval overtime
	every := bot.config.ReminderAnnounceInterval.Duration

	announcements := time.Until(future) / every
	if announcements <= 0 {
		future := time.Until(future)
		if tsk == endAuction && future < time.Duration(time.Second*300)  && future > time.Duration(time.Second * 180) {
			// make a reminder announcement after 2 minutes
			return reminderAnnouncement, time.Now().Add(2 * time.Minute)
		}
	}

	if tsk == endAuction && time.Until(future) < time.Duration(time.Second*83) {
		bot.runningCountDown = true
		// start countdown if there is almost 60 seconds left till the end
		return startCountDown, time.Time{}
	}

	nearFuture := future.Add(-announcements * every)
	switch tsk {
	case endAuction:
		// make a reminder announcement soon
		return reminderAnnouncement, nearFuture
	default:
		log.Print("unsupported task to subSchedule")
		return nothing, time.Time{}
	}
}

func (bot *Bot) perform(tsk task) {
	event := bot.currentAuction
	if event == nil {
		log.Print("failed to perform the scheduled task: no current auction")
		return
	}

	noctx := &Context{}
	switch tsk {
	case reminderAnnouncement:
		bot.Send(noctx, "yell", "html", fmt.Sprintf(`Auction ends @%s`, niceTime(bot.auctionEndTime.UTC())))
	case startCountDown:
		bot.runningCountDown = true

		bot.Send(noctx, "yell", "html", `<i>Alright cats and kitties. Get all your cryptos together and be prepared. I'm going to count down from 20 to 1. The last highest bid in the countdown will be approved by me as winning bid. Good luck!</i>`)
		for i := bot.config.CountdownFrom; i>0; i-- {
			time.Sleep(time.Second * 3)
			bot.Send(noctx, "yell", "text", fmt.Sprintf("%v", i))
		}
		bot.runningCountDown = false
		// end auction
		event.Ended = true
		event.exists = true
		bot.db.PutAuction(event)
		bot.currentAuction = nil

		bot.Reply(&Context{
			message: bot.lastBidMessage.UserMsg,
			User: &User{ID: bot.lastBidMessage.UserMsg.From.ID},
		}, fmt.Sprintf(`Congratulations %s, you've won the auction! Please contact @erichkaestner for the details on how to claim your new Legendary Kitty.`, bot.lastBidMessage.UserMsg.From.FirstName))
	default:
		log.Printf("unsupported task to perform: %v", tsk)
	}
}

func (bot *Bot) maintain() {
	bot.rescheduleChan = make(chan int)
	defer func() {
		close(bot.rescheduleChan)
	}()

	bot.bidChan = make(chan int, 2000)
	var timer *time.Timer
	for {

		tsk, future := bot.subSchedule()

		if timer == nil {
			timer = time.NewTimer(time.Until(future))
		} else {
			timer.Reset(time.Until(future))
		}

		select {
		case <-timer.C:
			bot.perform(tsk)
		case <-bot.rescheduleChan:
			if !timer.Stop() {
				<-timer.C
			}
		}

	}
}

// Cause a reschedule to happen. Call this if you modify events, so that the
// bot could wake itself up at correct times for automatic announcements and
// event starting/stopping.
func (bot *Bot) Reschedule() {
	bot.rescheduleChan <- 1
}
