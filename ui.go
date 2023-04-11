package main

import (
	"github.com/gdamore/tcell/v2"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/protosio/distributeddolt/db"
	"github.com/rivo/tview"
)

func uiUpdate(app *tview.Application, peerListView *tview.List, commitTreeRoot *tview.TreeNode, textView *tview.TextView, peerListChan chan peer.IDSlice, commitListChan chan []db.Commit, eventChan chan []byte) func() error {
	stopSignal := make(chan struct{})
	go func() {
		log.Info("Starting UI updater")
		for {
			select {
			case peerList := <-peerListChan:
				peerListView.Clear()
				for _, peer := range peerList {
					peerListView.AddItem(peer.ShortString(), "", 0, nil)
				}
				app.Draw()
			case event := <-eventChan:
				_, err := textView.Write(event)
				if err != nil {
					panic(err)
				}
				app.Draw()
			case commitList := <-commitListChan:
				commitTreeRoot.ClearChildren()
				for _, commit := range commitList {
					node := tview.NewTreeNode(commit.Hash)
					commitTreeRoot.AddChild(node)
				}
				app.Draw()
			case <-stopSignal:
				log.Info("Stopping ui updater")
				return
			}
		}
	}()
	stopper := func() error {
		stopSignal <- struct{}{}
		return nil
	}
	return stopper
}

func createUI(peerListChan chan peer.IDSlice, commitListChan chan []db.Commit, eventChan chan []byte) *tview.Application {
	var app = tview.NewApplication()
	var flex = tview.NewFlex()

	treeRoot := tview.NewTreeNode(".").SetColor(tcell.ColorRed)
	tree := tview.NewTreeView().SetRoot(treeRoot)
	tree.SetBorder(true).SetTitle("Commits")

	textView := tview.NewTextView()
	textView.SetBorder(true).SetTitle("Events")

	peerList := tview.NewList()
	peerList.SetBorder(true).SetTitle("Peers")

	flex.AddItem(peerList, 0, 1, false).
		AddItem(tree, 0, 1, false).
		AddItem(textView, 0, 2, false)

	uiUpdateStopper := uiUpdate(app, peerList, treeRoot, textView, peerListChan, commitListChan, eventChan)
	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyCtrlC {
			uiUpdateStopper()
			app.Stop()
		}
		return event
	})

	app.SetRoot(flex, true)
	app.EnableMouse(true)
	return app

}
