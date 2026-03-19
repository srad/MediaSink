package ws

import (
	"testing"

	"github.com/gorilla/websocket"
)

func TestWsDispatcherAddWsStoresOriginalPointer(t *testing.T) {
	dispatcher := wsDispatcher{}
	conn := &wsConnection{ws: &websocket.Conn{}}

	dispatcher.addWs(conn)

	if len(dispatcher.listeners) != 1 {
		t.Fatalf("expected 1 listener, got %d", len(dispatcher.listeners))
	}
	if dispatcher.listeners[0] != conn {
		t.Fatal("dispatcher stored a copied websocket connection instead of the original pointer")
	}
}

func TestWsDispatcherRmWsRemovesMatchingConnection(t *testing.T) {
	dispatcher := wsDispatcher{}
	first := &wsConnection{ws: &websocket.Conn{}}
	second := &wsConnection{ws: &websocket.Conn{}}

	dispatcher.addWs(first)
	dispatcher.addWs(second)
	dispatcher.rmWs(first.ws)

	if len(dispatcher.listeners) != 1 {
		t.Fatalf("expected 1 listener after removal, got %d", len(dispatcher.listeners))
	}
	if dispatcher.listeners[0] != second {
		t.Fatal("dispatcher removed the wrong websocket connection")
	}
}
