package authorizers

import (
	"errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cognitoidentityprovider"
	"github.com/dgrijalva/jwt-go"
	"github.com/go-gnss/ntrip/caster"
)

// Cognito is an implementation of a ntrip.caster Authorizer
type Cognito struct {
	UserPoolID string
	ClientID   string
	Cip        *cognitoidentityprovider.CognitoIdentityProvider
}

// NewCognitoAuthorizer constructs a Cognito object given the Cognito user pool and client IDs
func NewCognitoAuthorizer(userPoolID, clientID string) (auth Cognito) {
	auth.UserPoolID = userPoolID
	auth.ClientID = clientID
	auth.Cip = cognitoidentityprovider.New(session.Must(session.NewSession()))
	return auth
}

// Authorize parses Basic Auth from a request and authenticates against Cognito
func (auth Cognito) Authorize(conn *caster.Connection) (err error) {
	switch conn.Request.Method {
	case "GET":
		return nil // TODO: Implement list of Closed mountpoints for which a client needs authorized access

	case "POST":
		username, password, exists := conn.Request.BasicAuth()
		if !exists {
			return errors.New("Basic auth not provided")
		}

		params := cognitoidentityprovider.AdminInitiateAuthInput{
			AuthFlow: aws.String("ADMIN_NO_SRP_AUTH"),
			AuthParameters: map[string]*string{
				"USERNAME": aws.String(username),
				"PASSWORD": aws.String(password),
			},
			ClientId:   aws.String(auth.ClientID),
			UserPoolId: aws.String(auth.UserPoolID),
		}
		resp, err := auth.Cip.AdminInitiateAuth(&params)
		if err != nil {
			return err
		}

		if resp.AuthenticationResult == nil {
			return errors.New(*resp.ChallengeName)
		}

		token, _ := jwt.Parse(*resp.AuthenticationResult.IdToken, nil)
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			return errors.New("No claims in JWT")
		}

		if groups, exists := claims["cognito:groups"]; exists {
			for _, group := range groups.([]interface{}) {
				if group == "mount:"+conn.Request.URL.Path[1:] {
					return nil
				}
			}
		}

		return errors.New("Not authorized for Mountpoint")
	}

	// Not sure if it makes sense to return the ID token in a header
	// Usually you would have the auth endpoint be elsewhere and return the token in the body of the response, but we don't really have the luxury of palming it off
	//conn.Writer.Header().Set("Authorization", "Bearer " + *resp.AuthenticationResult.IdToken)
	return errors.New("Method not implemented")
}
