package gomux

import (
	"fmt"
	"math/rand"
	"time"
)

var (
	left = []string{
		"devoted",
		"dissipated",
		"extravagant",
		"faithful",
		"firm",
		"indulgent",
		"lavish",
		"loyal",
		"precious",
		"priceless",
		"profligate",
		"reckless",
		"shameless",
		"steadfast",
		"unrestrained",
		"valuable",
		"wild",
	}

	right = []string{
		"acura",
		"audi",
		"bentley",
		"bmw",
		"buick",
		"cadillac",
		"chevrolet",
		"chrysler",
		"dodge",
		"ferrari",
		"fiat",
		"ford",
		"gmc",
		"honda",
		"hyundai",
		"infiniti",
		"jaguar",
		"jeep",
		"kia",
		"lamborghini",
		"land",
		"lexus",
		"lincoln",
		"lotus",
		"maserati",
		"mazda",
		"mclaren",
		"mini",
		"mitsubishi",
		"nissan",
		"porsche",
		"rover",
		"scion",
		"subaru",
		"suzuki",
		"tesla",
		"toyota",
		"volkswagen",
		"volvo",
	}
)

type nameChecker func(chosen string) bool

func generateName(id string, checker nameChecker) string {
	rand.Seed(time.Now().UnixNano())
	chosen := fmt.Sprintf("%s_%s", left[rand.Intn(len(left))],
		right[rand.Intn(len(right))])
	// if the chosen name exists, just append the first 10 chars of id to it
	// and call it a day
	if checker(chosen) {
		return fmt.Sprintf("%s_%s", chosen, id[0:6])
	}
	// otherwise, return nice name here
	return chosen
}
