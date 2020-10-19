package handlers

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/go-redsync/redsync/v4"
	"net/http"
	"time"
)

func ElectionHandler(rs *redsync.Redsync, rds *redis.Client, c *gin.Context) {
	sid := c.Param("sid") // service ID
	rid := c.Param("rid") // round ID

	// create the corresponding mutex
	// mutex should last a while, to lower DB/network thrashing
	mutex := rs.NewMutex(fmt.Sprintf("gone/mutex/%s/%s", sid, rid), redsync.WithExpiry(60*time.Minute), redsync.WithTries(4))
	// give ourselves 10 seconds of grace to acquire the mutex
	ctx, _ := context.WithDeadline(c.Request.Context(), time.Now().Add(10*time.Second))

	// acquire the lock.  If we don't then respond in kind.
	if err := mutex.LockContext(ctx); err != nil {
		fmt.Printf("unable to acquire mutex for %s/%s: '%s'\n", sid, rid, err.Error())
		http.Error(c.Writer, http.StatusText(http.StatusNoContent), http.StatusNoContent)
		return
	}

	// we won this round.  Let's verify this wasn't a mis-call
	// e.g. client called us 3 days after the round was already over
	result := rds.Get(c.Request.Context(), fmt.Sprintf("gone/elected/%s/%s", sid, rid))
	if result.Err() == redis.Nil {
		// this is a new entry, we're the winner for the round
		setRes := rds.Set(c.Request.Context(), fmt.Sprintf("gone/elected/%s/%s", sid, rid), c.Request.RemoteAddr, 0)
		if setRes.Err() != nil {
			fmt.Println("unexpected error when asserting leadership: " + result.Err().Error())
			http.Error(c.Writer, http.StatusText(http.StatusInternalServerError)+result.Err().Error(), http.StatusInternalServerError)

			// unlock the mutex: we had it, but failed to record that we got it
			mutex.Unlock()
			return
		}
		c.Writer.WriteHeader(http.StatusAccepted)
	} else if result.Err() != nil {
		// if we got an error from this call, return an internal server error. this should be able to be retried
		fmt.Println("unexpected error when verifying leadership: " + result.Err().Error())
		http.Error(c.Writer, http.StatusText(http.StatusInternalServerError)+result.Err().Error(), http.StatusInternalServerError)
		return
	} else {
		// this is not a new entry. client called us long after the round ended
		http.Error(c.Writer, http.StatusText(http.StatusNoContent)+result.Val(), http.StatusNoContent)
		return
	}
}

func ElectionQueryHandler(_ *redsync.Redsync, rds *redis.Client, c *gin.Context) {
	sid := c.Param("sid") // service ID
	rid := c.Param("rid") // round ID

	result := rds.Get(c.Request.Context(), fmt.Sprintf("gone/elected/%s/%s", sid, rid))
	if result.Err() == redis.Nil {
		fmt.Printf("checked for round leadership, but no leader found! : %s/%s\n", sid, rid)
		http.Error(c.Writer, http.StatusText(http.StatusNoContent)+result.Val(), http.StatusNoContent)
	} else if result.Err() != nil {
		// if we got an error from this call, return an internal server error. this should be able to be retried
		fmt.Println("unexpected error when verifying leadership: " + result.Err().Error())
		http.Error(c.Writer, http.StatusText(http.StatusInternalServerError)+result.Err().Error(), http.StatusInternalServerError)
		return
	} else {
		c.Writer.WriteHeader(http.StatusOK)
		_, _ = c.Writer.WriteString(result.Val())
	}
}
