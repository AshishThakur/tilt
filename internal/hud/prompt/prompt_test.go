package prompt

import (
	"context"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/tilt-dev/tilt/internal/store"
	"github.com/tilt-dev/tilt/internal/testutils/bufsync"
	"github.com/tilt-dev/tilt/pkg/model"
)

const FakeURL = "http://localhost:10350/"

func TestOpenBrowser(t *testing.T) {
	f := newFixture()
	defer f.TearDown()

	f.prompt.OnChange(f.ctx, f.st)

	assert.Contains(t, f.out.String(), "(space) to open the browser")

	f.input.nextRune <- ' '
	assert.Equal(t, FakeURL, f.b.WaitForURL(t))
}

type fixture struct {
	ctx    context.Context
	cancel func()
	out    *bufsync.ThreadSafeBuffer
	st     *store.TestingStore
	b      *fakeBrowser
	input  *fakeInput
	prompt *TerminalPrompt
}

func newFixture() *fixture {
	ctx, cancel := context.WithCancel(context.Background())
	out := bufsync.NewThreadSafeBuffer()
	st := store.NewTestingStore()
	st.WithState(func(state *store.EngineState) {
		state.TerminalMode = store.TerminalModePrompt
	})
	i := &fakeInput{ctx: ctx, nextRune: make(chan rune)}
	b := &fakeBrowser{url: make(chan string)}
	openInput := OpenInput(func() (TerminalInput, error) { return i, nil })

	url, _ := url.Parse(FakeURL)

	prompt := NewTerminalPrompt(openInput, b.OpenURL, out, "localhost", model.WebURL(*url))
	return &fixture{
		ctx:    ctx,
		cancel: cancel,
		out:    out,
		st:     st,
		input:  i,
		b:      b,
		prompt: prompt,
	}
}

func (f *fixture) TearDown() {
	f.cancel()
}

type fakeInput struct {
	ctx      context.Context
	nextRune chan rune
}

func (i *fakeInput) Close() error { return nil }

func (i *fakeInput) ReadRune() (rune, error) {
	select {
	case r := <-i.nextRune:
		return r, nil
	case <-i.ctx.Done():
		return 0, i.ctx.Err()
	}
}

type fakeBrowser struct {
	url chan string
}

func (b *fakeBrowser) WaitForURL(t *testing.T) string {
	select {
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for url")
		return ""
	case url := <-b.url:
		return url
	}
}

func (b *fakeBrowser) OpenURL(url string) error {
	b.url <- url
	return nil
}
