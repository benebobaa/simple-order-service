package integration

func (s *authSuite) TestMe_RequiresValidToken() {
	noToken := doJSON("GET", "/v1/auth/me", nil, "")
	s.Equal(401, noToken.Code)
	s.Equal("UNAUTHORIZED", decodeError(s.T(), noToken).Error.Code)

	garbageToken := doJSON("GET", "/v1/auth/me", nil, "not-a-jwt")
	s.Equal(401, garbageToken.Code)
	s.Equal("UNAUTHORIZED", decodeError(s.T(), garbageToken).Error.Code)
}
