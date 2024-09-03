package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"log"
	"math"
	"math/rand"
	"net/http"
	"shinner/pkg/shinner"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/gorilla/websocket"
)

type App struct {
	shinner  *shinner.Shinner
	userID   string
	userName string
	path     []historyPath
	conn     *websocket.Conn
}

// Generic function to generate random numbers for both int and float64 types
func randValue[T int | float64](min, max T) T {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	switch v := any(min).(type) {
	case int:
		return T(v + r.Intn(int(any(max).(int)-v)))
	case float64:
		return T(v + r.Float64()*(any(max).(float64)-v))
	}
	return min
}

type historyPath struct {
	Lat    float64 `json:"lat"`
	Lon    float64 `json:"lon"`
	Radius float64 `json:"radius"`
	Shins  []shins `json:"shins"`
}

type shins struct {
	Lat    float64 `json:"lat"`
	Lon    float64 `json:"lon"`
	Amount int     `json:"amount"`
	Owner  bool    `json:"owner"`
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func TraverseGlobe(callback func(lat, lon, radius float64)) {
	log.Println("🌍 Starting to traverse the globe")
	radius := randValue(200.0, 800.0)                   // In kilometers
	earthCircumference := 40075.0                       // Circumference of the Earth in kilometers
	latitudeStep := radius / earthCircumference * 360.0 // Convert the distance to degrees of latitude
	longitudeStep := 2 * latitudeStep                   // Increase longitude step to space out the circles

	for lon := -180.0; lon <= 180.0; lon += longitudeStep {
		log.Println("🌍 Moving to longitude:", lon)

		if int(lon/longitudeStep)%2 == 0 {
			// Moving down from the North Pole to the South Pole
			for lat := 90.0; lat >= -90.0; lat -= latitudeStep {
				// Appliquer un décalage à gauche en fonction de la latitude
				longitudeCorrection := lon - (latitudeStep / math.Cos(lat*math.Pi/180.0))
				callback(lat, longitudeCorrection, radius)
			}
		} else {
			// Moving up from the South Pole to the North Pole
			for lat := -90.0; lat <= 90.0; lat += latitudeStep {
				// Appliquer un décalage à gauche en fonction de la latitude
				longitudeCorrection := lon - (latitudeStep / math.Cos(lat*math.Pi/180.0))
				callback(lat, longitudeCorrection, radius)
			}
		}
	}

	log.Println("🌍 Finished traversing the globe")
}

func (a *App) setHistory(h historyPath) {
	a.path = append(a.path, h)

	if a.conn != nil {
		a.conn.WriteJSON(h)
	}
}

func New(apiKey string) *App {
	return &App{
		path:    []historyPath{},
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
	ctx := context.Background()

	user, err := app.shinner.GetUser(ctx, app.userID)
	if err != nil {
		log.Fatalf("failed to get user: %v", err)
	}

	app.userName = user.GetUser.Username

	go func() {
		TraverseGlobe(func(lat, lon, radius float64) {
			h := historyPath{
				Lat:    lat,
				Lon:    lon,
				Radius: radius,
			}

			resp, err := app.shinner.GetNearbyShins(ctx, lat, lon, h.Radius)
			if err != nil {
				log.Println("Failed to get nearby shins:", err)
				return
			}

			for _, shin := range resp.GetNearbyShins.Shins {
				shinRecord := shins{
					Lat:    shin.Latitude,
					Lon:    shin.Longitude,
					Amount: shin.Amount,
				}

				if shin.FoundBy.Username == "" {
					// Collect the Shin
					if err := app.shinner.CollectShin(ctx, shinner.CollectShinInput{
						ID:     shin.ID,
						UserID: app.userID,
						Amount: shin.Amount,
					}); err != nil {
						log.Printf("❌ failed to collect shin with ID %s: %v\n", shin.ID, err)
					}
					sleepTime := randValue(1, 3)
					time.Sleep(time.Duration(sleepTime) * time.Second)
					log.Printf("💰 successfully collected Shin at http://localhost:8080/map with amount: %d\n", shin.Amount)
					shinRecord.Owner = true
				} else {
					log.Printf("💪 shin already collected by %s at http://localhost:8080/map \n", shin.FoundBy.Username)
					if shin.FoundBy.Username == app.userName {
						shinRecord.Owner = true
					} else {
						shinRecord.Owner = false
					}
				}

				h.Shins = append(h.Shins, shinRecord)
			}
			sleepTime := randValue(0, 2)
			time.Sleep(time.Duration(sleepTime) * time.Second)
			app.setHistory(h)
		})
	}()

	http.HandleFunc("/map", app.renderMapHandler)
	http.HandleFunc("/ws", app.wsHandler)

	app.displayInfoTable()

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}

func (a *App) displayInfoTable() {
	s := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render

	t := table.New()
	t.Headers("Name", "URL")
	t.Row("Live Map", s("http://localhost:8080/map "))
	t.Row("Email", flagEmail)
	t.Row("UserName", a.userName)
	t.Row("Authored by", "https://github.com/edouard-claude/shinner-bot ")
	fmt.Println(t.Render())
}

func (a *App) wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Failed to upgrade to WebSocket:", err)
		return
	}
	defer conn.Close()

	a.conn = conn

	if err := a.conn.WriteJSON(a.path); err != nil {
		log.Println("Failed to send path data:", err)
		return
	}

	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			log.Println("WebSocket connection closed:", err)
			break
		}
	}
}

// renderMapHandler renders the HTML page with the map and path data
func (a *App) renderMapHandler(w http.ResponseWriter, r *http.Request) {
	// Convert the paths to a JSON string for embedding in the HTML
	pathsJSON, err := json.Marshal(a.path)
	if err != nil {
		http.Error(w, "Failed to encode paths to JSON", http.StatusInternalServerError)
		return
	}

	// Render the HTML template
	tmpl := template.New("map")
	tmpl, err = tmpl.Parse(MapTemplate)
	if err != nil {
		http.Error(w, "Failed to parse template", http.StatusInternalServerError)
		return
	}

	data := struct {
		Paths template.JS
	}{
		Paths: template.JS(pathsJSON),
	}

	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, "Failed to execute template", http.StatusInternalServerError)
		return
	}
}

// MapTemplate is the HTML template for rendering the map
var MapTemplate = `
<!DOCTYPE html>
<html>
<head>
    <title>🗺️ Shinner Bot Live</title>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <link rel="stylesheet" href="https://unpkg.com/leaflet/dist/leaflet.css" />
    <script src="https://unpkg.com/leaflet/dist/leaflet.js"></script>
    <style>
		html, body { margin: 0; padding: 0; }
        #map { height: 100vh; }
    </style>
</head>
<body>
    <div id="map"></div>
    <script>
        var map = L.map('map').setView([0, 0], 2); // Center map on the equator, zoom level 2

        L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
            maxZoom: 18,
            attribution: '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors'
        }).addTo(map);

        // WebSocket for real-time updates
        var ws = new WebSocket("ws://" + window.location.host + "/ws");

        ws.onmessage = function(event) {
            var path = JSON.parse(event.data);

            // Add circle to the map
            L.circle([path.lat, path.lon], {
                color: 'blue',
                fillOpacity: 0.02,				
                radius: path.radius * 1000 // Convert radius to meters				
            }).bindTooltip("Radius: " + path.radius + " km").addTo(map);

			// add shins to the map
			path?.shins?.forEach(function(shin) {
				L.circle([shin.lat, shin.lon], {
					color: shin.owner ? 'green' : 'red',
					fillColor: shin.owner ? 'green' : 'red',
					fillOpacity: 0.5,
					radius: 600
				}).addTo(map);
			});
        };
    </script>
</body>
</html>
`
