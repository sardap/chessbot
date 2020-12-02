package db

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/sardap/chessbot/chess"
	"github.com/sardap/chessbot/env"
)

//Instance DB Connection instance
type Instance struct {
	db *redis.Client
}

//Connect connect
func (i *Instance) Connect() {
	for {
		i.db = redis.NewClient(&redis.Options{
			Addr:     env.RedisAddress,
			Password: env.RedisPassword,
			DB:       env.RedisDB,
		})

		if err := i.db.Ping(context.TODO()).Err(); err != nil {
			fmt.Printf("Unable to connect %v trying again in 5 seconds\n", err)
			time.Sleep(time.Duration(5) * time.Second)
			continue
		}

		break
	}
}

//DeleteGame Deletes a game from the DB
func (i *Instance) DeleteGame(g *chess.Game) error {
	return i.db.Del(
		context.TODO(),
		g.ID(),
	).Err()
}

//SaveGame saves a game under the active section of the DB
func (i *Instance) SaveGame(g *chess.Game) error {
	bytes, err := json.Marshal(*g)
	if err != nil {
		return err
	}

	return i.db.Set(
		context.TODO(), g.ID(), bytes,
		time.Duration(24)*time.Hour,
	).Err()
}

//GetGame gets a game from the DB
func (i *Instance) GetGame(id string) (*chess.Game, error) {
	ctx := context.TODO()

	res := i.db.Get(ctx, id)
	if res.Err() != nil {
		return nil, res.Err()
	}

	var result chess.Game
	err := json.Unmarshal([]byte(res.Val()), &result)

	result.ProcessMoves()

	return &result, err
}

//ArchiveGame archives a game in the DB
func (i *Instance) ArchiveGame(g *chess.Game) error {
	byts, err := json.Marshal(*g)
	if err != nil {
		return err
	}

	var b bytes.Buffer
	gz := gzip.NewWriter(&b)

	if _, err := gz.Write(byts); err != nil {
		return err
	}
	gz.Close()

	id := fmt.Sprintf("%s_%s", time.Now().UTC().Format("2006:01:02-15:04:05"), g.ID())
	i.db.Set(context.TODO(), id, b.Bytes(), 0)

	return nil
}
