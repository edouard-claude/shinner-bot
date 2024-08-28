package shinner

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/machinebox/graphql"
)

const (
	apiShinnerUrl = "https://shinner-backend-production-amd4bgjgpq-ez.a.run.app/api/graphql"
	userAent      = "shinner/1050 CFNetwork/1496.0.7 Darwin/23.5.0"
	apolloClient  = "shinner-app-prod-ios"
)

type Shinner struct {
	client *graphql.Client
	token  string
	uA     string
	aC     string
	userID string
	apiKey string
}

func New(apiKey string) *Shinner {
	return &Shinner{
		client: graphql.NewClient(apiShinnerUrl),
		apiKey: apiKey,
		uA:     userAent,
		aC:     apolloClient,
	}
}

type LoginResponse struct {
	DisplayName  string `json:"displayName"`
	Email        string `json:"email"`
	ExpiresIn    string `json:"expiresIn"`
	IDToken      string `json:"idToken"`
	Kind         string `json:"kind"`
	LocalID      string `json:"localId"`
	RefreshToken string `json:"refreshToken"`
	Registered   bool   `json:"registered"`
}

func (s *Shinner) Login(email, password string) (*LoginResponse, error) {
	url := fmt.Sprintf("https://www.googleapis.com/identitytoolkit/v3/relyingparty/verifyPassword?key=%s", s.apiKey)

	payload := map[string]interface{}{
		"clientType":        "CLIENT_TYPE_IOS",
		"email":             email,
		"password":          password,
		"returnSecureToken": true,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal login payload: %v", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create login request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-client-version", "iOS/FirebaseSDK/10.20.0/FirebaseCore-iOS")
	req.Header.Set("x-ios-bundle-identifier", "com.dahlsjoo.shinner")
	req.Header.Set("accept-language", "en")
	req.Header.Set("user-agent", "FirebaseAuth.iOS/10.20.0 com.dahlsjoo.shinner/1.9.0 iPhone/17.5.1 hw/iPhone15_4")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute login request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("login request failed with status: %s", resp.Status)
	}

	var loginResp LoginResponse
	if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
		return nil, fmt.Errorf("failed to decode login response: %v", err)
	}

	s.token = loginResp.IDToken

	return &loginResp, nil
}

type RefreshTokenRequest struct {
	GrantType    string `json:"grantType"`
	RefreshToken string `json:"refreshToken"`
}

type RefreshTokenResponse struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    string `json:"expires_in"`
	IDToken      string `json:"id_token"`
	ProjectID    string `json:"project_id"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	UserID       string `json:"user_id"`
}

func (s *Shinner) RefreshToken(refreshToken RefreshTokenRequest) (*RefreshTokenResponse, error) {
	url := fmt.Sprintf("https://securetoken.googleapis.com/v1/token?key=%s", s.apiKey)

	jsonData, err := json.Marshal(refreshToken)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal refresh token payload: %v", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create refresh token request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-client-version", "iOS/FirebaseSDK/10.20.0/FirebaseCore-iOS")
	req.Header.Set("x-ios-bundle-identifier", "com.dahlsjoo.shinner")
	req.Header.Set("accept-language", "en")
	req.Header.Set("user-agent", "FirebaseAuth.iOS/10.20.0 com.dahlsjoo.shinner/1.9.0 iPhone/17.5.1 hw/iPhone15_4")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute refresh token request: %v", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("refresh token request failed with status: %s", resp.Status)
	}

	var refreshTokenResp RefreshTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&refreshTokenResp); err != nil {
		return nil, fmt.Errorf("failed to decode refresh token response: %v", err)
	}

	s.token = refreshTokenResp.IDToken

	return &refreshTokenResp, nil
}

type GetNearbyShins struct {
	GetNearbyShins struct {
		Shins []struct {
			Amount  int   `json:"amount,omitempty"`
			Found   int64 `json:"found,omitempty"`
			FoundBy struct {
				Avatar struct {
					Sm any `json:"sm,omitempty"`
				} `json:"avatar,omitempty"`
				ID       string `json:"id,omitempty"`
				Username string `json:"username,omitempty"`
			} `json:"foundBy,omitempty"`
			ID        string  `json:"id,omitempty"`
			Latitude  float64 `json:"latitude,omitempty"`
			Longitude float64 `json:"longitude,omitempty"`
		} `json:"shins,omitempty"`
	} `json:"getNearbyShins,omitempty"`
}

func (s *Shinner) GetNearbyShins(ctx context.Context, latitude, longitude, radius float64) (*GetNearbyShins, error) {
	req := graphql.NewRequest(`
		query GetNearbyShins($input: GetNearbyShinsInput!) {
			getNearbyShins(input: $input) {
				shins {
					id
					amount
					latitude
					longitude
					found
					foundBy {
						username
						id
						avatar {
							sm
						}
					}
				}
			}
		}
	`)

	req.Var("input", map[string]interface{}{
		"latitude":  latitude,
		"longitude": longitude,
		"radius":    radius,
	})

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.token))
	req.Header.Set("User-Agent", s.uA)
	req.Header.Set("apollographql-client-name", s.aC)

	var resp GetNearbyShins
	if err := s.client.Run(ctx, req, &resp); err != nil {
		return nil, fmt.Errorf("failed to execute query GetNearbyShins: %v", err)
	}

	return &resp, nil
}

type GetSpots struct {
	GetSpots struct {
		Edges []struct {
			Node struct {
				ID       string `json:"id,omitempty"`
				Location struct {
					City        string `json:"city,omitempty"`
					Coordinates struct {
						Latitude  float64 `json:"latitude,omitempty"`
						Longitude float64 `json:"longitude,omitempty"`
					} `json:"coordinates,omitempty"`
					Country string `json:"country,omitempty"`
				} `json:"location,omitempty"`
				Media []struct {
					Urls struct {
						Carousel string `json:"carousel,omitempty"`
					} `json:"urls,omitempty"`
				} `json:"media,omitempty"`
				Meta struct {
					Created   int64 `json:"created,omitempty"`
					CreatedBy struct {
						Avatar   any    `json:"avatar,omitempty"`
						ID       string `json:"id,omitempty"`
						Name     string `json:"name,omitempty"`
						Username string `json:"username,omitempty"`
					} `json:"createdBy,omitempty"`
					Crew      any `json:"crew,omitempty"`
					Event     any `json:"event,omitempty"`
					UpdatedBy any `json:"updatedBy,omitempty"`
				} `json:"meta,omitempty"`
				Status string   `json:"status,omitempty"`
				Tags   []string `json:"tags,omitempty"`
				Title  string   `json:"title,omitempty"`
			} `json:"node,omitempty"`
		} `json:"edges,omitempty"`
		PageInfo struct {
			EndCursor   string `json:"endCursor,omitempty"`
			HasNextPage bool   `json:"hasNextPage,omitempty"`
			StartCursor string `json:"startCursor,omitempty"`
		} `json:"pageInfo,omitempty"`
		TotalCount int `json:"totalCount,omitempty"`
	} `json:"getSpots,omitempty"`
}

func (s *Shinner) GetSpots(ctx context.Context, last int) (*GetSpots, error) {
	req := graphql.NewRequest(`
		query GetSpots($last: Int, $before: String) {
			getSpots(last: $last, before: $before) {
				totalCount
				pageInfo {
					startCursor
					endCursor
					hasNextPage
				}
				edges {
					node {
						id
						title
						tags
						location {
							city
							country
							coordinates {
								latitude
								longitude
							}
						}
						media {
							urls {
								carousel
							}
						}
						meta {
							created
							createdBy {
								id
								username
								name
								avatar {
									original
									sm
								}
							}
							event {
								id
								title
								eventTitle
								challengeTitle
								challengeId
								firstPlace
								secondPlace
								thirdPlace
							}
							crew {
								id
								name
							}
							updatedBy {
								id
								username
								name
								avatar {
									original
									sm
								}
							}
						}
						status
					}
				}
			}
		}
	`)

	req.Var("last", last)

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.token))
	req.Header.Set("User-Agent", s.uA)
	req.Header.Set("apollographql-client-name", s.aC)

	var resp GetSpots
	if err := s.client.Run(ctx, req, &resp); err != nil {
		return nil, fmt.Errorf("failed to execute query GetSpots: %v", err)
	}

	return &resp, nil
}

type CollectShinInput struct {
	ID     string `json:"id"`
	UserID string `json:"userId"`
	Amount int    `json:"amount"`
}

func (s *Shinner) CollectShin(ctx context.Context, input CollectShinInput) error {
	req := graphql.NewRequest(`
		mutation CollectShin($input: CollectShinInput!) {
			collectShin(input: $input)
		}
	`)

	req.Var("input", input)

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.token))
	req.Header.Set("User-Agent", s.uA)
	req.Header.Set("apollographql-client-name", s.aC)

	var resp struct {
		Data struct {
			CollectShin bool `json:"collectShin"`
		} `json:"data"`
	}

	if err := s.client.Run(ctx, req, &resp); err != nil {
		return fmt.Errorf("failed to execute mutation CollectShin: %v", err)
	}

	return nil
}

type GetUser struct {
	Data struct {
		GetUser struct {
			Typename string `json:"__typename"`
			Avatar   struct {
				Typename string `json:"__typename"`
				Original string `json:"original"`
				Sm       any    `json:"sm"`
			} `json:"avatar"`
			Crews       []any  `json:"crews"`
			DateOfBirth string `json:"dateOfBirth"`
			Email       string `json:"email"`
			Events      []any  `json:"events"`
			ID          string `json:"id"`
			Ilike       any    `json:"ilike"`
			Instagram   string `json:"instagram"`
			Name        string `json:"name"`
			Scene       struct {
				Typename   string `json:"__typename"`
				City       string `json:"city"`
				Coordinate struct {
					Typename  string  `json:"__typename"`
					Latitude  float64 `json:"latitude"`
					Longitude float64 `json:"longitude"`
				} `json:"coordinate"`
				Country   string `json:"country"`
				Formatted string `json:"formatted"`
				Region    string `json:"region"`
				Slug      string `json:"slug"`
			} `json:"scene"`
			Shins          int    `json:"shins"`
			Sponsors       string `json:"sponsors"`
			Stance         string `json:"stance"`
			StartedSkating string `json:"startedSkating"`
			Tiktok         any    `json:"tiktok"`
			Topics         []struct {
				Typename string `json:"__typename"`
				Title    string `json:"title"`
				Topic    string `json:"topic"`
			} `json:"topics"`
			Username string `json:"username"`
			Youtube  any    `json:"youtube"`
		} `json:"getUser"`
	} `json:"data"`
}

func (s *Shinner) GetUser(ctx context.Context, id string) (*GetUser, error) {
	req := graphql.NewRequest(`
		query GetUser($id: ID!) {
			getUser(id: $id) {
				id
				avatar {
					sm
					original
					__typename
				}
				crews {
					id
					__typename
				}
				email
				events {
					id
					__typename
				}
				name
				username
				shins
				stance
				ilike
				startedSkating
				dateOfBirth
				sponsors
				instagram
				tiktok
				youtube
				scene {
					city
					coordinate {
						latitude
						longitude
						__typename
					}
					country
					formatted
					region
					slug
					__typename
				}
				topics {
					title
					topic
					__typename
				}
				__typename
			}
		}
	`)

	req.Var("id", id)

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.token))
	req.Header.Set("User-Agent", s.uA)
	req.Header.Set("apollographql-client-name", s.aC)

	var resp GetUser
	if err := s.client.Run(ctx, req, &resp); err != nil {
		return nil, fmt.Errorf("failed to execute query GetUser: %v", err)
	}

	return &resp, nil
}
