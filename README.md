# Chirpy - Another Twitter Copy
A fully-fledged social media API server built from scratch in Go. A Twitter-like platform where users can post short messages called "chirps". Hit the project with a star if you find it useful.

## Motivation
This project demonstrates building a complete REST API from the ground up using Go's standard library and minimal dependencies. It showcases modern Go web development practices including proper project structure, authentication, database integration, and clean architecture patterns.

## Goal
The goal with Chirpy is to provide a complete example of a production-ready Go web API that includes:

- User authentication and authorization with JWT tokens
- RESTful API design with proper HTTP methods and status codes
- Database integration with PostgreSQL using SQLC
- Secure password hashing with bcrypt
- Refresh token management
- Webhook integration for payment processing
- Clean, maintainable code structure
- Comprehensive error handling

## ⚙️ Installation

### Prerequisites
- Go 1.21 or higher
- PostgreSQL database
- Git

### Setup
1. Clone the repository:
```bash
git clone https://github.com/dmitriy-zverev/chirpy.git
cd chirpy
```

2. Install dependencies:
```bash
go mod download
```

3. Set up your environment variables by creating a `.env` file:
```bash
DB_URL=postgres://username:password@localhost:5432/chirpy?sslmode=disable
JWT_SECRET=your-secret-key-here
POLKA_KEY=your-polka-webhook-key
PLATFORM=dev
PORT=8080
JWT_EXPIRATION_TIME=1h
```

4. Run database migrations:
```bash
# Apply your SQL schema files to your PostgreSQL database
```

5. Start the server:
```bash
go run main.go
```

## 🚀 Quick Start

### Creating a User
```bash
curl -X POST http://localhost:8080/api/users \
  -H "Content-Type: application/json" \
  -d '{"email": "user@example.com", "password": "securepassword"}'
```

### User Login
```bash
curl -X POST http://localhost:8080/api/login \
  -H "Content-Type: application/json" \
  -d '{"email": "user@example.com", "password": "securepassword"}'
```

### Creating a Chirp
```bash
curl -X POST http://localhost:8080/api/chirps \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -d '{"body": "This is my first chirp!"}'
```

### Getting All Chirps
```bash
curl http://localhost:8080/api/chirps
```

## 📁 Project Structure

```
chirpy/
├── main.go                 # Application entry point with refactored structure
├── internal/
│   ├── auth/              # Authentication and JWT handling
│   ├── database/          # Database models and queries (SQLC generated)
│   └── handlers/          # HTTP handlers and API configuration
├── sql/
│   ├── queries/           # SQL queries for SQLC
│   └── schema/            # Database schema migrations
├── assets/                # Static files
└── README.md
```

## 🔧 API Endpoints

### Authentication
- `POST /api/users` - Create a new user
- `PUT /api/users` - Update user information
- `POST /api/login` - User login
- `POST /api/refresh` - Refresh JWT token
- `POST /api/revoke` - Revoke refresh token

### Chirps
- `GET /api/chirps` - Get all chirps (supports filtering and sorting)
- `GET /api/chirps/{chirpID}` - Get a specific chirp
- `POST /api/chirps` - Create a new chirp (requires authentication)
- `DELETE /api/chirps/{chirpID}` - Delete a chirp (requires authentication)

### Admin
- `GET /admin/metrics` - View server metrics
- `POST /admin/reset` - Reset database (dev environment only)

### Health
- `GET /api/healthz` - Health check endpoint

### Webhooks
- `POST /api/polka/webhooks` - Polka payment webhook

## 🔒 Authentication

Chirpy uses JWT (JSON Web Tokens) for authentication. After logging in, include the token in the Authorization header:

```
Authorization: Bearer YOUR_JWT_TOKEN
```

Refresh tokens are also supported for maintaining long-term sessions.

## 🗄️ Database

The project uses PostgreSQL with SQLC for type-safe database queries. The database schema includes:

- Users table with hashed passwords
- Chirps table with user relationships
- Refresh tokens for authentication
- Chirpy Red premium user status

## 🧪 Testing

Run the tests:
```bash
go test ./...
```

## 📝 Configuration

The application can be configured using environment variables:

- `DB_URL` - PostgreSQL connection string
- `JWT_SECRET` - Secret key for JWT signing
- `POLKA_KEY` - API key for Polka webhooks
- `PLATFORM` - Environment (dev/prod)
- `PORT` - Server port (default: 8080)
- `JWT_EXPIRATION_TIME` - JWT token expiration duration

## 🚀 Deployment

The application is designed to be easily deployable to various platforms. Ensure all environment variables are properly set in your production environment.

## 🛠️ Technologies Used

- **Go** - Programming language
- **PostgreSQL** - Database
- **SQLC** - Type-safe SQL query generation
- **JWT** - Authentication tokens
- **bcrypt** - Password hashing
- **Standard Library** - Minimal external dependencies

## 💬 Contact

- GitHub: [@dmitriy-zverev](https://github.com/dmitriy-zverev)
- Submit an issue here on GitHub

## 👏 Contributing

Contributions are welcome! Please feel free to submit a Pull Request. For major changes, please open an issue first to discuss what you would like to change.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📄 License

This project is fully open source. Feel free to use it as you wish.
