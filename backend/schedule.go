package main

import (
	"encoding/json"
	"net/url"
	"slices"
)

type Notification struct {
	ID        string `json:"id"`
	Kind      string `json:"kind"`
	Reference string `json:"reference"`
	Recipient string `json:"recipient,omitempty"`
	Title     string `json:"title"`
	Detail    string `json:"detail"`
	At        string `json:"at"`
	Href      string `json:"href"`
	Popup     bool   `json:"popup"`
}

func trainingVisible(s Training) bool { return s.Published == nil || *s.Published }

func applicantName(d *Data, ap *Application) string {
	if ap.Guest != nil {
		return ap.Guest.Name
	}
	if account, ok := d.Accounts[ap.User]; ok {
		return account.Name
	}
	return "حساب محذوف"
}

func notify(d *Data, item Notification) {
	item.ID = nonce()
	d.Notifications = append(d.Notifications, item)
}

func visibleNotification(d *Data, a Account, n Notification) bool {
	switch n.Kind {
	case "interview":
		ap := appFor(d, n.Reference)
		return n.Recipient == a.ID && programOpen(d, "trainer") && can(a, "trainer.view") &&
			ap != nil && ap.Appointment == n.At
	case "lecture":
		s := trainingFor(d, n.Reference)
		return programOpen(d, "trainer") && can(a, "trainer.view") && s != nil && trainingVisible(*s) && s.At == n.At
	case "bulletin":
		i := familyIndex(d, n.Reference)
		return programOpen(d, "family") && can(a, "family.view") && i >= 0 && !d.Bulletins[i].Hidden
	}
	return false
}

func notificationView(d *Data, a Account) []map[string]any {
	items := []map[string]any{}
	for _, n := range slices.Backward(d.Notifications) {
		if !visibleNotification(d, a, n) {
			continue
		}
		items = append(items, map[string]any{"id": n.ID, "kind": n.Kind, "title": n.Title, "detail": n.Detail,
			"at": n.At, "href": n.Href, "popup": n.Popup, "read": d.NotificationReads[a.ID][n.ID]})
		if len(items) == 50 {
			break
		}
	}
	return items
}

func readNotification(d *Data, a Account, raw json.RawMessage) error {
	var in struct{ ID string }
	if err := payload(raw, &in); err != nil {
		return err
	}
	for _, n := range d.Notifications {
		if n.ID != in.ID || !visibleNotification(d, a, n) {
			continue
		}
		if d.NotificationReads == nil {
			d.NotificationReads = map[string]map[string]bool{}
		}
		if d.NotificationReads[a.ID] == nil {
			d.NotificationReads[a.ID] = map[string]bool{}
		}
		d.NotificationReads[a.ID][n.ID] = true
		return nil
	}
	return missing()
}

type CalendarEvent struct {
	ID     string `json:"id"`
	Kind   string `json:"kind"`
	Title  string `json:"title"`
	Detail string `json:"detail"`
	At     string `json:"at"`
}

func calendarView(d *Data, a Account) []CalendarEvent {
	events := []CalendarEvent{}
	for _, ap := range d.Applications {
		if ap.Appointment == "" {
			continue
		}
		c := campaignFor(d, ap.Campaign)
		assigned := c != nil && can(a, "trainer.evaluate") && includes(c.Panel, a.ID)
		if ap.User == a.ID || can(a, "trainer.decide") || assigned {
			events = append(events, CalendarEvent{ap.ID, "interview", "مقابلة · " + ap.Topic, applicantName(d, &ap), ap.Appointment})
		}
	}
	for _, s := range d.Trainings {
		if !trainingVisible(s) && !can(a, "trainer.session") {
			continue
		}
		if ap := appFor(d, s.Application); ap != nil {
			kind := "lecture"
			if !trainingVisible(s) {
				kind = "draft"
			}
			events = append(events, CalendarEvent{s.ID, kind, s.Title, applicantName(d, ap), s.At})
		}
	}
	return events
}

func announceTraining(d *Data, s *Training, ap *Application) {
	notify(d, Notification{Kind: "lecture", Reference: s.ID, Title: "محاضرة · " + s.Title,
		Detail: "يقدّمها " + applicantName(d, ap), At: s.At, Href: "/knowledge?tab=calendar", Popup: s.At[:10] >= today()})
}

func announceBulletin(d *Data, b Bulletin) {
	notify(d, Notification{Kind: "bulletin", Reference: b.ID, Title: "نشرة جديدة · " + b.Title,
		Detail: "Deraah Family", At: b.At, Href: "/family?tab=current&issue=" + url.QueryEscape(b.ID), Popup: true})
}
