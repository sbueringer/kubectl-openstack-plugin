package mattermost

import "testing"

func Test(t *testing.T) {

	New().SendMessage("test text")

}
