package handlers

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/go-redsync/redsync/v4"
	"net/http"
)

func CompletionHandler(_ *redsync.Redsync, rds *redis.Client, c *gin.Context) {
	sid := c.Param("sid") // service ID
	rid := c.Param("rid") // round ID

	// verify someone else didn't mark as completed
	// this is unexpected as it's only expected to be called once. therefore no mutex is used
	result := rds.Get(c.Request.Context(), fmt.Sprintf("gone/completed/%s/%s", sid, rid))
	if result.Err() == redis.Nil {
		// happy path. record our completion
		setRes := rds.Set(c.Request.Context(), fmt.Sprintf("gone/completed/%s/%s", sid, rid), c.Request.RemoteAddr, 0)

		if setRes.Err() != nil {
			fmt.Println("unexpected error when recording completion: " + result.Err().Error())
			http.Error(c.Writer, http.StatusText(http.StatusInternalServerError)+result.Err().Error(), http.StatusInternalServerError)
			return
		}
		c.Writer.WriteHeader(http.StatusOK)
	} else if result.Err() != nil {
		// if we got an error from this call, return an internal server error. this can be retried
		fmt.Println("unexpected error when verifying leadership: " + result.Err().Error())
		http.Error(c.Writer, http.StatusText(http.StatusInternalServerError)+result.Err().Error(), http.StatusInternalServerError)
		return
	} else {
		// this is not a new entry. another leader(!!!) has already recorded success
		http.Error(c.Writer, http.StatusText(http.StatusBadRequest)+result.Val(), http.StatusBadRequest)
		return
	}
}

func CompletionQueryHandler(_ *redsync.Redsync, rds *redis.Client, c *gin.Context) {
	sid := c.Param("sid") // service ID
	rid := c.Param("rid") // round ID

	result := rds.Get(c.Request.Context(), fmt.Sprintf("gone/completed/%s/%s", sid, rid))
	if result.Err() == redis.Nil {
		fmt.Printf("checked for round completion, but no completer found! : %s/%s\n", sid, rid)
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
