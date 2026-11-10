package main

import (
	"deraah/trainer/pulse"
	"net/http"
)

func pulseActor(a Account) pulse.Actor {
	role := "viewer"
	if a.System || a.Voice == "admin" {
		role = "admin"
	} else if can(a, "voice.campaign") || can(a, "voice.plan") {
		role = "analyst"
	}
	return pulse.Actor{
		User:    pulse.User{ID: a.ID, Email: a.Email, Name: a.Name, Role: role, Active: a.Status == "active"},
		CanEdit: can(a, "voice.campaign"), CanResults: can(a, "voice.results"),
		CanPlan: can(a, "voice.plan"), CanReview: can(a, "voice.review"),
		CanPublish: can(a, "voice.publish"),
	}
}

func (s *server) resolvePulseActor(r *http.Request) (pulse.Actor, error) {
	d, err := s.read(r.Context())
	if err != nil {
		return pulse.Actor{}, err
	}
	a, err := signedIn(&d, r)
	if err != nil {
		return pulse.Actor{}, err
	}
	if !can(a, "voice.view") {
		return pulse.Actor{}, forbidden()
	}
	return pulseActor(a), nil
}
