package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math"
	"math/rand"
	"shinner/pkg/shinner"
	"strings"
	"time"
)

type App struct {
	shinner *shinner.Shinner
	userID  string
}

const (
	radius      = 2500.0 // rayon de chaque sphère en kilomètres
	earthRadius = 6371.0 // rayon de la Terre en kilomètres
)

func randInt(min, max int) int {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return min + r.Intn(max-min)
}

func randFloat(min, max float64) float64 {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return min + r.Float64()*(max-min)
}

func clamp(value, min, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func (a *App) TraverseEarthToCollectShins(ctx context.Context) error {
	radius := randFloat(1000, radius)
	// Calculer les étapes en latitude et longitude
	stepLat := 2 * radius / earthRadius * 180 / math.Pi
	stepLon := 2 * radius / earthRadius * 180 / math.Pi / math.Cos(0) // approximatif pour l'équateur

	for lat := -90.0; lat <= 90.0; lat += stepLat {
		for lon := -180.0; lon <= 180.0; lon += stepLon {
			// Simuler un mouvement de "pinch" aléatoire
			pinchZoom := randFloat(0.8, 1.2)   // léger zoom ou dézoom
			radius *= pinchZoom                // ajuster le radius
			radius = clamp(radius, 1000, 3000) // Limiter le radius dans une plage raisonnable
			log.Println("🤏 pinching with zoom factor:", pinchZoom, "📍 new radius:", radius)

			// Recalculer les étapes en fonction du nouveau radius
			stepLat = 2 * radius / earthRadius * 180 / math.Pi
			stepLon = 2 * radius / earthRadius * 180 / math.Pi / math.Cos(0)

			// Introduire une légère pause aléatoire pour simuler la lecture ou l'interaction humaine
			sleepDuration := time.Duration(randInt(1, 5)) * time.Second
			log.Println("🌍 lat:", lat, "lon:", lon, "😴 sleep:", sleepDuration)
			time.Sleep(sleepDuration)

			// Introduire un décalage aléatoire dans le mouvement pour éviter les trajectoires parfaitement régulières
			latOffset := randFloat(-stepLat/10, stepLat/10)
			lonOffset := randFloat(-stepLon/10, stepLon/10)

			// Appliquer les décalages tout en respectant les limites de latitude et longitude
			lat = clamp(lat+latOffset, -90.0, 90.0)
			lon = clamp(lon+lonOffset, -180.0, 180.0)

			resp, err := a.shinner.GetNearbyShins(ctx, lat, lon, radius)
			if err != nil {
				if strings.Contains(err.Error(), "Too many shins") {
					log.Println("🚫 too many shins error skipped")
					continue
				} else {
					return fmt.Errorf("❌ failed to get nearby shins at lat: %f, lon: %f, err: %v", lat, lon, err)
				}
			}

			if len(resp.GetNearbyShins.Shins) > 0 {
				for _, shin := range resp.GetNearbyShins.Shins {
					// Vérifier si le Shin n'a pas été collecté
					if shin.FoundBy.Username == "" {
						// Collecter le Shin
						if err := a.shinner.CollectShin(ctx, shinner.CollectShinInput{
							ID:     shin.ID,
							UserID: a.userID,
							Amount: shin.Amount,
						}); err != nil {
							return fmt.Errorf("❌ failed to collect shin with ID %s: %v", shin.ID, err)
						}

						log.Printf("💰 successfully collected Shin at https://maps.google.com/?q=%f,%f with amount: %d\n", shin.Latitude, shin.Longitude, shin.Amount)
					} else {
						log.Printf("💪 shin already collected by %s at https://maps.google.com/?q=%f,%f \n", shin.FoundBy.Username, shin.Latitude, shin.Longitude)
					}
					log.Println("😴 sleeping for a while before the next Shin")
					time.Sleep(time.Duration(randInt(1, 5)) * time.Second)
				}
			} else {
				log.Println("📍 no Shin found at this location")
			}
		}
	}
	return nil
}

func New(apiKey string) *App {
	return &App{
		shinner: shinner.New(apiKey),
	}
}

var (
	flagApiKey,
	flagEmail,
	flagPassword string
)

func init() {
	flag.StringVar(&flagApiKey, "api", "", "API key")
	flag.StringVar(&flagEmail, "email", "", "Email")
	flag.StringVar(&flagPassword, "password", "", "Password")
	flag.Parse()

	if flagApiKey == "" {
		log.Fatal("api key is required")
		flag.Usage()
	}

	if flagEmail == "" {
		log.Fatal("email is required")
		flag.Usage()
	}

	if flagPassword == "" {
		log.Fatal("password is required")
		flag.Usage()
	}
}

func main() {
	app := New(flagApiKey)

	ctx := context.Background()

	dataLogin, err := app.shinner.Login(flagEmail, flagPassword)
	if err != nil {
		log.Fatalf("failed to login: %v", err)
	}

	log.Println("🔑 logged in successfully")

	dataRefresh, err := app.shinner.RefreshToken(shinner.RefreshTokenRequest{
		GrantType:    "refresh_token",
		RefreshToken: dataLogin.RefreshToken,
	})
	if err != nil {
		log.Fatalf("failed to refresh token: %v", err)
	}

	log.Println("🔄 token refreshed successfully")

	app.userID = dataRefresh.UserID

	if err := app.TraverseEarthToCollectShins(ctx); err != nil {
		log.Printf("Error while traversing the Earth: %v\n", err)
	}
}
