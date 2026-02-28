package main

// JSON-friendly WebAuthn options (base64url strings for ArrayBuffer fields).

type rpEntity struct {
	Name string `json:"name"`
	ID   string `json:"id,omitempty"`
}

type userEntity struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
}

type pubKeyCredParam struct {
	Type string `json:"type"`
	Alg  int    `json:"alg"`
}

type publicKeyCredentialDescriptor struct {
	Type       string   `json:"type"`
	ID         string   `json:"id"`
	Transports []string `json:"transports,omitempty"`
}

type authenticatorSelection struct {
	AuthenticatorAttachment string `json:"authenticatorAttachment,omitempty"`
	ResidentKey             string `json:"residentKey,omitempty"`
	RequireResidentKey      bool   `json:"requireResidentKey,omitempty"`
	UserVerification        string `json:"userVerification,omitempty"`
}

type publicKeyCredentialCreationOptions struct {
	Challenge              string                        `json:"challenge"`
	RP                     rpEntity                      `json:"rp"`
	User                   userEntity                    `json:"user"`
	PubKeyCredParams       []pubKeyCredParam             `json:"pubKeyCredParams"`
	Timeout                int                           `json:"timeout,omitempty"`
	Attestation            string                        `json:"attestation,omitempty"`
	AuthenticatorSelection authenticatorSelection         `json:"authenticatorSelection,omitempty"`
	ExcludeCredentials     []publicKeyCredentialDescriptor `json:"excludeCredentials,omitempty"`
}

type publicKeyCredentialRequestOptions struct {
	Challenge        string                        `json:"challenge"`
	RPID             string                        `json:"rpId,omitempty"`
	Timeout          int                           `json:"timeout,omitempty"`
	UserVerification string                        `json:"userVerification,omitempty"`
	AllowCredentials []publicKeyCredentialDescriptor `json:"allowCredentials,omitempty"`
}

