package api

type Keeper interface {
	Keep() int
}

func Use(k Keeper) int {
	return k.Keep()
}
