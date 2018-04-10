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
	auction := bot.db.GetCurrentAuction()
	if auction == nil {
		return nothing, time.Time{}
	}

	if auction.EndTime.Valid {
		bot.auctionEndTime = auction.EndTime.Time
		return endAuction, auction.EndTime.Time.Add(time.Second*-200)
	}

	return nothing, time.Time{}

}

// Returns a more detailed version than `schedule()`
// of what to do next (including announcements).
func (bot *Bot) subSchedule() (task, time.Time) {
	fmt.Println(bot.runningCountDown)
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
		fmt.Println("this reminder")
		future := time.Until(future)
		if tsk == endAuction && future < time.Duration(time.Second*300)  && future > time.Duration(time.Second * 180) {
			// make a reminder announcement after 2 minutes
			return reminderAnnouncement, time.Now().Add(2 * time.Minute)
		}
	}

	if tsk == endAuction && time.Until(future) < time.Duration(time.Second*103) {
		bot.runningCountDown = true
		// start countdown if there is almost 100 seconds left till the end
		return startCountDown, time.Time{}
	}

	nearFuture := future.Add(-announcements * every)
	fmt.Println(nearFuture)
	fmt.Println("no, this reminder")
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
	event := bot.db.GetCurrentAuction()
	if event == nil {
		log.Print("failed to perform the scheduled task: no current auction")
		return
	}

	noctx := &Context{}
	switch tsk {
	case reminderAnnouncement:
		bot.Send(noctx, "yell", "html", fmt.Sprintf(`Auction ends @%s`, niceTime(bot.auctionEndTime.UTC())))
	case startCountDown:
		for i := bot.config.CountdownFrom; i>bot.config.ResettingCountdownFrom; i-- {
			bot.Send(noctx, "yell", "text", fmt.Sprintf("%v", i))
			time.Sleep(time.Second * 4)
		}
		for i := bot.config.ResettingCountdownFrom; i > 0; i-- {
			select {
			// if a bid was placed reset the counter
			case <-bot.bidChan:
				if i > 8 {
					i = 8
				} else {
					bot.Send(noctx, "yell", "text", fmt.Sprintf("%v", i))
					time.Sleep(time.Second * 2)
				}
			default:
				bot.Send(noctx, "yell", "text", fmt.Sprintf("%v", i))
				time.Sleep(time.Second * 2)
			}
		}

		bot.Reply(bot.lastBidMessage, `Please PM @erichkaestner`)
		bot.runningCountDown = false
		fmt.Println(bot.db.EndAuction())
	default:
		log.Printf("unsupported task to perform: %v", tsk)
	}
}

func (bot *Bot) maintain() {
	bot.rescheduleChan = make(chan int)
	defer func() {
		close(bot.rescheduleChan)
	}()

	bot.bidChan = make(chan int, 200)
	var timer *time.Timer
	for {

		tsk, future := bot.subSchedule()

		fmt.Println(tsk)
		fmt.Println(future)
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

		fmt.Println("finally")
	}
}

// Cause a reschedule to happen. Call this if you modify events, so that the
// bot could wake itself up at correct times for automatic announcements and
// event starting/stopping.
func (bot *Bot) Reschedule() {
	bot.rescheduleChan <- 1
}
