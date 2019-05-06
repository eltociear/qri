package cron

import (
	"context"
	"testing"
	"time"

	"github.com/qri-io/ioes"
	"github.com/qri-io/iso8601"
)

func mustRepeatingInterval(s string) iso8601.RepeatingInterval {
	ri, err := iso8601.ParseRepeatingInterval(s)
	if err != nil {
		panic(err)
	}
	return ri
}

func TestCronDataset(t *testing.T) {
	updateCount := 0
	job := &Job{
		Name:        "b5/libp2p_node_count",
		Type:        JTDataset,
		Periodicity: mustRepeatingInterval("R/P1W"),
	}

	factory := func(outer context.Context) RunJobFunc {
		return func(ctx context.Context, streams ioes.IOStreams, job *Job) error {
			switch job.Type {
			case JTDataset:
				updateCount++
				// ds.Commit.Timestamp = time.Now()
				return nil
			}
			t.Fatalf("runner called with invalid job: %v", job)
			return nil
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*500)
	defer cancel()

	cron := NewCronInterval(&MemJobStore{}, &MemJobStore{}, factory, time.Millisecond*50)
	if err := cron.Schedule(ctx, job); err != nil {
		t.Fatal(err)
	}

	if err := cron.Start(ctx); err != nil {
		t.Fatal(err)
	}

	<-ctx.Done()

	expectedUpdateCount := 1
	if expectedUpdateCount != updateCount {
		t.Errorf("update ran wrong number of times. expected: %d, got: %d", expectedUpdateCount, updateCount)
	}
}

func TestCronShellScript(t *testing.T) {
	pdci := DefaultCheckInterval
	defer func() { DefaultCheckInterval = pdci }()
	DefaultCheckInterval = time.Millisecond * 50

	updateCount := 0

	job := &Job{
		Name:        "foo.sh",
		Type:        JTShellScript,
		Periodicity: mustRepeatingInterval("R/P1W"),
	}

	// scriptRunner := LocalShellScriptRunner("testdata")
	factory := func(outer context.Context) RunJobFunc {
		return func(ctx context.Context, streams ioes.IOStreams, job *Job) error {
			switch job.Type {
			case JTShellScript:
				updateCount++
				// return scriptRunner(ctx, streams, job)
				return nil
			}
			t.Fatalf("runner called with invalid job: %v", job)
			return nil
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*500)
	defer cancel()

	cron := NewCron(&MemJobStore{}, &MemJobStore{}, factory)
	if err := cron.Schedule(ctx, job); err != nil {
		t.Fatal(err)
	}

	if err := cron.Start(ctx); err != nil {
		t.Fatal(err)
	}

	<-ctx.Done()

	expectedUpdateCount := 1
	if expectedUpdateCount != updateCount {
		t.Errorf("update ran wrong number of times. expected: %d, got: %d", expectedUpdateCount, updateCount)
	}
}

func TestCronHTTP(t *testing.T) {
	s := &MemJobStore{}
	l := &MemJobStore{}

	factory := func(context.Context) RunJobFunc {
		return func(ctx context.Context, streams ioes.IOStreams, job *Job) error {
			return nil
		}
	}

	cliCtx := context.Background()
	cli := HTTPClient{Addr: ":7897"}
	if err := cli.Ping(); err != ErrUnreachable {
		t.Error("expected ping to server that is off to return ErrUnreachable")
	}

	cr := NewCron(s, l, factory)
	// TODO (b5) - how do we keep this from being a leaking goroutine?
	go cr.ServeHTTP(":7897")

	time.Sleep(time.Millisecond * 100)
	if err := cli.Ping(); err != nil {
		t.Errorf("expected ping to active server to not fail. got: %s", err)
	}

	jobs, err := cli.Jobs(cliCtx, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(jobs) != 0 {
		t.Error("expected 0 jobs")
	}

	dsJob := &Job{
		Name:         "b5/libp2p_node_count",
		Type:         JTDataset,
		Periodicity:  mustRepeatingInterval("R/P1W"),
		LastRunStart: time.Date(2019, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	if err = cli.Schedule(cliCtx, dsJob); err != nil {
		t.Fatal(err.Error())
	}

	jobs, err = cli.Jobs(cliCtx, 0, 0)
	if err != nil {
		t.Fatal(err.Error())
	}

	if len(jobs) != 1 {
		t.Error("expected len of jobs to equal 1")
	}

	_, err = cli.Job(cliCtx, jobs[0].Name)
	if err != nil {
		t.Fatal(err.Error())
	}

	if err := cli.Unschedule(cliCtx, dsJob.Name); err != nil {
		t.Fatal(err)
	}

	jobs, err = cli.Jobs(cliCtx, 0, 0)
	if err != nil {
		t.Fatal(err.Error())
	}

	if len(jobs) != 0 {
		t.Error("expected len of jobs to equal 0")
	}
}