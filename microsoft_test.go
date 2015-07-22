package microsoft

import (
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type AuthRequestMock struct{ mock.Mock }

type TokenResponseMock struct{ mock.Mock }

func (m *AuthRequestMock) GetAccessToken() TokenResponse {
	ret := m.Mock.Called()
	return ret.Get(0).(TokenResponse)
}

func (m *TokenResponseMock) Translate() (string, error) {
	ret := m.Mock.Called()
	return ret.Get(0).(string), ret.Error(1)
}

func (m *TokenResponseMock) TranslateArray() ([]string, error) {
	ret := m.Mock.Called()
	return ret.Get(0).([]string), ret.Error(1)
}

func (m *TokenResponseMock) CheckTimeout() bool {
	ret := m.Mock.Called()
	return ret.Bool(0)
}

func makeValidTokenResponse() TokenResponse {
	now := time.Now()
	expires_in, _ := strconv.ParseInt("600", 10, 0)
	// 10 min
	exp_time := now.Add(time.Duration(expires_in) * time.Second)

	return TokenResponse{
		AccessToken: "s3cr3t0k3n",
		ExpiresIn:   "600",
		Timeout:     exp_time,
	}
}

func TestGetToken(t *testing.T) {
	mockObj := new(AuthRequestMock)
	tokenResponse := makeValidTokenResponse()
	mockObj.On("GetAccessToken").Return(tokenResponse)

	tr := GetAccessToken(mockObj)

	mockObj.AssertExpectations(t)

	assert := assert.New(t)
	assert.Equal("s3cr3t0k3n", tr.AccessToken)
	assert.Equal("600", tr.ExpiresIn)
	assert.False(tr.CheckTimeout())
}

func TestTranslateText(t *testing.T) {
	mck := new(TokenResponseMock)

	mck.On("CheckTimeout").Return(false)
	mck.On("Translate").Return("um", nil)

	text, _ := TranslateText(mck)

	mck.AssertExpectations(t)

	assert := assert.New(t)
	assert.Equal("um", text)
}

func TestTranslateArrayText(t *testing.T) {
	mck := new(TokenResponseMock)

	mck.On("CheckTimeout").Return(false)
	mck.On("TranslateArray").Return([]string{"um dois", "livro de fotos"}, nil)

	text, _ := TranslateTexts(mck)

	mck.AssertExpectations(t)

	assert := assert.New(t)
	assert.Equal("um dois", text[0])
	assert.Equal("livro de fotos", text[1])
}

func TestTranslateWithTokenExpired(t *testing.T) {
	mck := new(TokenResponseMock)

	mck.On("CheckTimeout").Return(true)
	text, err := TranslateText(mck)

	mck.AssertExpectations(t)

	assert := assert.New(t)
	assert.Equal("", text)
	assert.EqualError(err, "Access token is invalid, please get new token")

}
