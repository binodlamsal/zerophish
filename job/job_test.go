package job

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewAndGetAndParams(t *testing.T) {
	params := []string{"foo", "bar"}
	j := New(params)
	assert.NotNil(t, j)
	assert.NotNil(t, Get(j.id))
	assert.Equal(t, params, j.Params().([]string))
}

func TestStart(t *testing.T) {
	var wg sync.WaitGroup
	j := New([]string{"foo", "bar"})

	wg.Add(1)

	j.Start(func(job *Job) {
		defer wg.Done()
		t.Log("Started a job")
		time.Sleep(1 * time.Second)
		job.Progress <- 50
		time.Sleep(1 * time.Second)
		j.Errors <- errors.New("Dummy error")
		time.Sleep(1 * time.Second)
		job.Progress <- 100
		job.Done <- true
	})

	wg.Wait()
	assert.Equal(t, 100, j.progress)
	assert.Len(t, j.errors, 1)
	time.Sleep(9 * time.Second)
	assert.Nil(t, Get(j.id))
}
