// Copyright (c) 2015 Spinpunch, Inc. All Rights Reserved.
// See License.txt for license information.

package api

import (
	"github.com/gorilla/websocket"
	"github.com/mattermost/platform/model"
	"github.com/mattermost/platform/store"
	"github.com/mattermost/platform/utils"
	"net/http"
	"testing"
	"time"
)

func TestSocket(t *testing.T) {
	Setup()

	url := "ws://localhost:" + utils.Cfg.ServiceSettings.Port + "/api/v1/websocket"
	team := &model.Team{Name: "Name", Domain: "z-z-" + model.NewId() + "a", Email: "test@nowhere.com", Type: model.TEAM_OPEN}
	team = Client.Must(Client.CreateTeam(team)).Data.(*model.Team)

	user1 := &model.User{TeamId: team.Id, Email: model.NewId() + "corey@test.com", FullName: "Corey Hulen", Password: "pwd"}
	user1 = Client.Must(Client.CreateUser(user1, "")).Data.(*model.User)
	store.Must(Srv.Store.User().VerifyEmail(user1.Id))
	Client.LoginByEmail(team.Domain, user1.Email, "pwd")

	channel1 := &model.Channel{DisplayName: "Test Web Scoket 1", Name: "a" + model.NewId() + "a", Type: model.CHANNEL_OPEN, TeamId: team.Id}
	channel1 = Client.Must(Client.CreateChannel(channel1)).Data.(*model.Channel)

	channel2 := &model.Channel{DisplayName: "Test Web Scoket 2", Name: "a" + model.NewId() + "a", Type: model.CHANNEL_OPEN, TeamId: team.Id}
	channel2 = Client.Must(Client.CreateChannel(channel2)).Data.(*model.Channel)

	header1 := http.Header{}
	header1.Set(model.HEADER_AUTH, "BEARER "+Client.AuthToken)

	c1, _, err := websocket.DefaultDialer.Dial(url, header1)
	if err != nil {
		t.Fatal(err)
	}

	user2 := &model.User{TeamId: team.Id, Email: model.NewId() + "corey@test.com", FullName: "Corey Hulen", Password: "pwd"}
	user2 = Client.Must(Client.CreateUser(user2, "")).Data.(*model.User)
	store.Must(Srv.Store.User().VerifyEmail(user2.Id))
	Client.LoginByEmail(team.Domain, user2.Email, "pwd")

	header2 := http.Header{}
	header2.Set(model.HEADER_AUTH, "BEARER "+Client.AuthToken)

	c2, _, err := websocket.DefaultDialer.Dial(url, header2)
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(300 * time.Millisecond)
	Client.Must(Client.JoinChannel(channel1.Id))

	// Read the join channel message that gets generated
	var rmsg model.Message
	if err := c2.ReadJSON(&rmsg); err != nil {
		t.Fatal(err)
	}

	// Test sending message without a channelId
	m := model.NewMessage("", "", "", model.ACTION_TYPING)
	m.Add("RootId", model.NewId())
	m.Add("ParentId", model.NewId())

	c1.WriteJSON(m)

	if err := c2.ReadJSON(&rmsg); err != nil {
		t.Fatal(err)
	}

	if team.Id != rmsg.TeamId {
		t.Fatal("Ids do not match")
	}

	if m.Props["RootId"] != rmsg.Props["RootId"] {
		t.Fatal("Ids do not match")
	}

	// Test sending messsage to Channel you have access to
	m = model.NewMessage("", channel1.Id, "", model.ACTION_TYPING)
	m.Add("RootId", model.NewId())
	m.Add("ParentId", model.NewId())

	c1.WriteJSON(m)

	if err := c2.ReadJSON(&rmsg); err != nil {
		t.Fatal(err)
	}

	if team.Id != rmsg.TeamId {
		t.Fatal("Ids do not match")
	}

	if m.Props["RootId"] != rmsg.Props["RootId"] {
		t.Fatal("Ids do not match")
	}

	// Test sending message to Channel you *do not* have access too
	m = model.NewMessage("", channel2.Id, "", model.ACTION_TYPING)
	m.Add("RootId", model.NewId())
	m.Add("ParentId", model.NewId())

	c1.WriteJSON(m)

	go func() {
		if err := c2.ReadJSON(&rmsg); err != nil {
			t.Fatal(err)
		}

		t.Fatal(err)
	}()

	time.Sleep(2 * time.Second)

	hub.Stop(team.Id)

}

func TestZZWebSocketTearDown(t *testing.T) {
	// *IMPORTANT* - Kind of hacky
	// This should be the last function in any test file
	// that calls Setup()
	// Should be in the last file too sorted by name
	time.Sleep(2 * time.Second)
	TearDown()
}
