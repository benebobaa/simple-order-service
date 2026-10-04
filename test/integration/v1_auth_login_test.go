package integration

import "github.com/benebobaa/simple-order-service/internal/auth"

func (s *authSuite) TestLogin_Success() {
	registerUserWithEmail(s.T(), "login@example.com")

	rec := doJSON("POST", "/v1/auth/login", map[string]any{
		"email":    "Login@Example.com",
		"password": "s3cret-pass",
	}, "")

	s.Require().Equal(200, rec.Code)
	resp := decodeData[auth.Response](s.T(), rec)
	s.NotEmpty(resp.Token)
	s.Equal("login@example.com", resp.User.Email)
}

func (s *authSuite) TestLogin_RejectsWrongPasswordAndUnknownEmail() {
	registerUserWithEmail(s.T(), "login@example.com")

	wrongPassword := doJSON("POST", "/v1/auth/login", map[string]any{
		"email":    "login@example.com",
		"password": "wrong-password",
	}, "")
	s.Require().Equal(401, wrongPassword.Code)
	s.Equal("INVALID_CREDENTIALS", decodeError(s.T(), wrongPassword).Error.Code)

	unknownEmail := doJSON("POST", "/v1/auth/login", map[string]any{
		"email":    "nobody@example.com",
		"password": "s3cret-pass",
	}, "")
	s.Require().Equal(401, unknownEmail.Code)
	s.Equal("INVALID_CREDENTIALS", decodeError(s.T(), unknownEmail).Error.Code)
}
