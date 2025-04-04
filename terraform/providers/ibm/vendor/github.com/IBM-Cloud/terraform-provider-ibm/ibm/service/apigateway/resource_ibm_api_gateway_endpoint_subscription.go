// Copyright IBM Corp. 2017, 2021 All Rights Reserved.
// Licensed under the Mozilla Public License v2.0

package apigateway

import (
	"fmt"
	"strings"

	"github.com/IBM-Cloud/terraform-provider-ibm/ibm/conns"
	"github.com/IBM-Cloud/terraform-provider-ibm/ibm/validate"
	apigatewaysdk "github.com/IBM/apigateway-go-sdk/apigatewaycontrollerapiv1"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func ResourceIBMApiGatewayEndpointSubscription() *schema.Resource {

	return &schema.Resource{
		Create:             resourceIBMApiGatewayEndpointSubscriptionCreate,
		Read:               resourceIBMApiGatewayEndpointSubscriptionGet,
		Update:             resourceIBMApiGatewayEndpointSubscriptionUpdate,
		Delete:             resourceIBMApiGatewayEndpointSubscriptionDelete,
		Importer:           &schema.ResourceImporter{},
		Exists:             resourceIBMApiGatewayEndpointSubscriptionExists,
		DeprecationMessage: "This service is deprecated.",
		Schema: map[string]*schema.Schema{
			"artifact_id": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Endpoint ID",
			},
			"client_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Subscription Id, API key that is used to create subscription",
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Subscription name",
			},
			"type": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validate.ValidateAllowedStringValues([]string{"external", "internal"}),
				Description:  "Subscription type. Allowable values are external, internal",
			},
			"client_secret": {
				Type:          schema.TypeString,
				Optional:      true,
				Sensitive:     true,
				ConflictsWith: []string{"generate_secret"},
				Description:   "Client Sercret of a Subscription",
			},
			"generate_secret": {
				Type:          schema.TypeBool,
				Optional:      true,
				ConflictsWith: []string{"client_secret"},
				Description:   "Indicates if Client Sercret has to be autogenerated",
			},
			"secret_provided": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Indicates if client secret is provided to subscription or not",
			},
		},
	}
}
func resourceIBMApiGatewayEndpointSubscriptionCreate(d *schema.ResourceData, meta interface{}) error {
	sess, err := meta.(conns.ClientSession).BluemixSession()
	if err != nil {
		return err
	}
	endpointservice, err := meta.(conns.ClientSession).APIGateway()
	if err != nil {
		return err
	}
	payload := &apigatewaysdk.CreateSubscriptionOptions{}

	oauthtoken := sess.Config.IAMAccessToken
	oauthtoken = strings.Replace(oauthtoken, "Bearer ", "", -1)
	payload.Authorization = &oauthtoken

	artifactID := d.Get("artifact_id").(string)
	payload.ArtifactID = &artifactID

	var clientID string
	if c, ok := d.GetOk("client_id"); ok && c != nil {
		clientID = c.(string)
		payload.ClientID = &clientID
	}
	var name string
	if v, ok := d.GetOk("name"); ok && v != nil {
		name = v.(string)
		payload.Name = &name
	}
	var shareType string
	if v, ok := d.GetOk("type"); ok && v != nil {
		shareType = v.(string)
		if shareType == "internal" {
			shareType = "bluemix"
		}
		payload.Type = &shareType
	}
	var clientSecret string
	if v, ok := d.GetOk("client_secret"); ok && v != nil {
		clientSecret = v.(string)
		payload.ClientSecret = &clientSecret
	}
	var generateSecret bool
	if g, ok := d.GetOk("generate_secret"); ok && g != nil {
		generateSecret = g.(bool)
		payload.GenerateSecret = &generateSecret
	}

	result, response, err := endpointservice.CreateSubscription(payload)
	if err != nil {
		return fmt.Errorf("[ERROR] Error creating Subscription: %s %s", err, response)
	}
	d.SetId(fmt.Sprintf("%s//%s", *result.ArtifactID, *result.ClientID))

	return resourceIBMApiGatewayEndpointSubscriptionGet(d, meta)
}

func resourceIBMApiGatewayEndpointSubscriptionGet(d *schema.ResourceData, meta interface{}) error {
	sess, err := meta.(conns.ClientSession).BluemixSession()
	if err != nil {
		return err
	}
	endpointservice, err := meta.(conns.ClientSession).APIGateway()
	if err != nil {
		return err
	}

	parts := d.Id()
	partslist := strings.Split(parts, "//")
	if len(partslist) < 2 {
		return fmt.Errorf("[ERROR] Incorrect ID %s: Id should be a combination of artifactID//clientID", d.Id())
	}
	artifactID := partslist[0]
	clientID := partslist[1]

	oauthtoken := sess.Config.IAMAccessToken
	oauthtoken = strings.Replace(oauthtoken, "Bearer ", "", -1)

	payload := apigatewaysdk.GetSubscriptionOptions{
		ArtifactID:    &artifactID,
		ID:            &clientID,
		Authorization: &oauthtoken,
	}
	result, response, err := endpointservice.GetSubscription(&payload)
	if err != nil {
		if response != nil && response.StatusCode == 404 {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("[ERROR] Error Getting Subscription: %s\n%s", err, response)
	}
	d.Set("artifact_id", result.ArtifactID)
	d.Set("client_id", result.ClientID)
	if *result.Type == "bluemix" {
		*result.Type = "internal"
	}
	d.Set("type", result.Type)
	if result.Name != nil {
		d.Set("name", result.Name)
	}
	d.Set("secret_provided", result.SecretProvided)
	return nil
}

func resourceIBMApiGatewayEndpointSubscriptionUpdate(d *schema.ResourceData, meta interface{}) error {
	sess, err := meta.(conns.ClientSession).BluemixSession()
	if err != nil {
		return err
	}
	endpointservice, err := meta.(conns.ClientSession).APIGateway()
	if err != nil {
		return err
	}
	payload := &apigatewaysdk.UpdateSubscriptionOptions{}

	parts := d.Id()
	partslist := strings.Split(parts, "//")
	artifactID := partslist[0]
	clientID := partslist[1]

	oauthtoken := sess.Config.IAMAccessToken
	oauthtoken = strings.Replace(oauthtoken, "Bearer ", "", -1)
	payload.Authorization = &oauthtoken

	payload.ID = &clientID
	payload.NewClientID = &clientID

	payload.ArtifactID = &artifactID
	payload.NewArtifactID = &artifactID

	name := d.Get("name").(string)
	payload.NewName = &name

	update := false

	if d.HasChange("name") {
		name := d.Get("name").(string)
		payload.NewName = &name
		update = true
	}
	if d.HasChange("client_secret") {
		clientSecret := d.Get("client_secret").(string)
		secretpayload := &apigatewaysdk.AddSubscriptionSecretOptions{
			Authorization: &oauthtoken,
			ArtifactID:    &artifactID,
			ID:            &clientID,
			ClientSecret:  &clientSecret,
		}
		_, SecretResponse, err := endpointservice.AddSubscriptionSecret(secretpayload)
		if err != nil {
			return fmt.Errorf("[ERROR] Error Adding Secret to Subscription: %s,%s", err, SecretResponse)
		}
	}
	if update {
		_, response, err := endpointservice.UpdateSubscription(payload)
		if err != nil {
			return fmt.Errorf("[ERROR] Error updating Subscription: %s,%s", err, response)
		}
	}
	return resourceIBMApiGatewayEndpointSubscriptionGet(d, meta)
}

func resourceIBMApiGatewayEndpointSubscriptionDelete(d *schema.ResourceData, meta interface{}) error {
	sess, err := meta.(conns.ClientSession).BluemixSession()
	if err != nil {
		return err
	}
	endpointservice, err := meta.(conns.ClientSession).APIGateway()
	if err != nil {
		return err
	}
	parts := d.Id()
	partslist := strings.Split(parts, "//")
	artifactID := partslist[0]
	clientID := partslist[1]

	oauthtoken := sess.Config.IAMAccessToken
	oauthtoken = strings.Replace(oauthtoken, "Bearer ", "", -1)

	payload := apigatewaysdk.DeleteSubscriptionOptions{
		ArtifactID:    &artifactID,
		ID:            &clientID,
		Authorization: &oauthtoken,
	}
	response, err := endpointservice.DeleteSubscription(&payload)
	if err != nil {
		if response != nil && response.StatusCode == 404 {
			return nil
		}
		return fmt.Errorf("[ERROR] Error deleting Subscription: %s\n%s", err, response)
	}
	d.SetId("")

	return nil
}

func resourceIBMApiGatewayEndpointSubscriptionExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	sess, err := meta.(conns.ClientSession).BluemixSession()
	if err != nil {
		return false, err
	}
	endpointservice, err := meta.(conns.ClientSession).APIGateway()
	if err != nil {
		return false, err
	}
	parts := d.Id()
	partslist := strings.Split(parts, "//")
	if len(partslist) < 2 {
		return false, fmt.Errorf("[ERROR] Incorrect ID %s: Id should be a combination of artifactID//clientID", d.Id())
	}
	artifactID := partslist[0]
	clientID := partslist[1]

	oauthtoken := sess.Config.IAMAccessToken
	oauthtoken = strings.Replace(oauthtoken, "Bearer ", "", -1)

	payload := apigatewaysdk.GetSubscriptionOptions{
		ArtifactID:    &artifactID,
		ID:            &clientID,
		Authorization: &oauthtoken,
	}
	_, response, err := endpointservice.GetSubscription(&payload)
	if err != nil {
		if response != nil && response.StatusCode == 404 {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
