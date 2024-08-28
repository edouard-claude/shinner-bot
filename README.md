# Shinner Bot

Shinner Bot is a Go application designed to traverse the Earth and collect
virtual "Shins" from different locations using the Shinner API.

## Features

- Traverse the globe and collect Shins within a defined radius.
- Randomize the search radius with slight zoom in/out effects to simulate a more
  natural search pattern.
- Automatically handles "too many shins" errors.
- Logs collected Shins with their geographical coordinates.
- Refreshes API tokens as needed to maintain a valid session.

## Installation

### Prerequisites

- Go 1.16 or later
- Shinner API key
- Shinner account credentials (email and password)

### Clone the Repository

```bash
git clone https://github.com/your-username/shinner-bot.git
cd shinner-bot
```

### Usage

Run the Shinner Bot by providing your Shinner API key, email, and password as
flags:

```bash
./shinner-bot -api shinner_api_key -email your_email -password your_password
```

### How It Works

- **Initialization**: The bot initializes with the provided API key and logs in
  using the provided email and password.
- **Token Refresh**: Once logged in, the bot refreshes the token to ensure
  session validity.
- **Traverse the Earth**: The bot begins traversing the Earth, searching for
  Shins within a randomized radius. The radius is dynamically adjusted during
  the traversal to simulate natural human behavior.
- **Shin Collection**: If Shins are found within the current location, the bot
  attempts to collect them, logging the details of each successful collection.
- **Error Handling**: The bot gracefully handles common errors, such as
  encountering too many Shins at a single location, by skipping those locations
  and continuing the traversal.

### Logging

The bot provides detailed logs for every step, including:

- The latitude and longitude of the current search.
- The radius used for the search.
- Details of any collected Shins, including a link to view them on Google Maps.
- Any errors encountered during the traversal and collection process.

### Contributing

If you'd like to contribute to Shinner Bot, please fork the repository and
submit a pull request.

### License

This project is licensed under the MIT License. See the LICENSE file for
details.
