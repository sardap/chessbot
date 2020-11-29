# Chess bot

A discord bot for playing chess over discord!

*Note*: currently there is no rule checks for any units so you need to make sure you make legal moves.

[Click here to join your server](https://discord.com/api/oauth2/authorize?client_id=782413879862493235&permissions=0&scope=bot)

## Building
Just run go build or build the docker image

## Running 
Refer to env/env.go to see what env vars you need set

## Using
`cb help` will print out all commands regex's (gross)
`cb start {@TARGET_PLAYER_HERE}` will start a new game with a player
`cb move {@TARGET_PLAYER_HERE} {FROM} {TO}` will move piece from coordinate to coordinate
`cb resign {@TARGET_PLAYER_HERE}` will concede a game 