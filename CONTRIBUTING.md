# Contributing to YUKI-MONGODB

Thank you for your interest in contributing to **YUKI-MONGODB**! 🍃

## How to Contribute

1. **Fork the Repository**:
   Click the Fork button at the top right of this repository.

2. **Clone your fork**:
   ```bash
   git clone https://github.com/<your-username>/YUKI-MONGODB.git
   cd YUKI-MONGODB
   ```

3. **Create a branch**:
   ```bash
   git checkout -b feature/my-new-feature
   ```

4. **Develop and Test**:
   - Ensure you have Go 1.22+ installed.
   - Run `go build -v ./cmd/engine` to compile the engine binary.
   - Test Docker build locally: `docker build -t yuki-mongo-test .`

5. **Commit your changes**:
   ```bash
   git commit -am 'feat: add awesome feature'
   ```

6. **Push to GitHub & Open a Pull Request**:
   ```bash
   git push origin feature/my-new-feature
   ```
   Open a PR against the `main` branch with a clear description of your changes.

## Code of Conduct

Be respectful, courteous, and constructive in discussions and reviews.
