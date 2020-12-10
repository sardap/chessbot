package env

import (
	"fmt"
	"image/jpeg"
	"os"
	"strconv"

	"github.com/pkg/errors"
)

var (
	//ImgQuality ImgQuality
	ImgQuality int
	//RedisAddress RedisAddress
	RedisAddress string
	//RedisPassword RedisPassword
	RedisPassword string
	//RedisDB redisDB
	RedisDB int
	//CmdPrefix the command prefix
	CmdPrefix string
)

func init() {
	imgQualityStr := os.Getenv("IMG_QUALITY")
	if imgQualityStr == "" {
		ImgQuality = jpeg.DefaultQuality
	}

	var err error
	ImgQuality, err = strconv.Atoi(imgQualityStr)
	if err != nil {
		panic(errors.Wrap(err, "Error reading IMG_QUALITY"))
	}

	RedisAddress = os.Getenv("REDIS_HOST")
	RedisPassword = os.Getenv("REDIS_PASSWORD")
	RedisDB, err = strconv.Atoi(os.Getenv("REDIS_DB"))
	if err != nil {
		panic(errors.Wrap(err, "Error reading REDIS_DB"))
	}

	CmdPrefix = os.Getenv("CMD_PREFIX")
	fmt.Printf("%s\n", CmdPrefix)
}
