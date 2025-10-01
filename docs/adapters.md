# Adapter Setup

## ZEP Adapter Setup

The ZEP Adapter just the Username and Password as well as the ZEP Endpoint as configuration parameters:

```yaml
source:
  adapter:
    type: "zep"
    calendar: "absences"
    config:
      username: testymctestface@inovex.de
      password: superSuperSecret1337
      endpoint: "https://zep.company.com/zep/sync/dav.php/calendars"
```

The ZEP adapter is only supported as a source.

## Outlook Adapter Setup
The Outlook calendar is synchronized via Microsoft Graph API. You will need to
[register an application on Azure](https://docs.microsoft.com/en-us/azure/active-directory/develop/quickstart-register-app).
The application needs the following permissions:

* `Calendar.ReadWrite`

The `User.read` permission should be assigned by default. To assign the `Calendar.ReadWrite` permission, click on "API Permissions" and add the permission to the "Microsoft Graph API".

You also need to setup a platform specific configuration. This can be done in the "Authentication" menu. Add a "mobile and desktop application" platform configuration and add `http://localhost/redirect` as a valid redirect uri.

![](../assets/azure_platform_config.png)

The Outlook adapter can be configured as a source using the tenant-id and client-id of the registered Azure app. Both
tenant-id and client-id can be found in the Azure portal "Overview" tab of the previously registered Azure app. If you want to use the adapter to access personal Microsoft accounts, you need to use the tenant `common`.

![](../assets/azure_app_ids.jpg)

```yaml
source:
  adapter:
    type: "outlook_http"
    calendar: "[base64-format string here]"
    config:
      tenantId: "[UUID-format string here]"
      clientId: "[UUID-format string here]"
```

To get your calendar ID, use the [Microsoft Graph Explorer](https://developer.microsoft.com/en-us/graph/graph-explorer) and query `GET https://graph.microsoft.com/v1.0/me/calendar`.


## Google Adapter Setup

For the setup of the Google adapter, an OAuth client has to be created. The client configuration is saved to `sync.yaml`
and on the first run, CalendarSync will open a browser window where you can authorize the application to use your calendar.
The setup of the OAuth client is done in the Google Cloud Console. Follow these steps:

+ Open the [Google Cloud Console](https://console.cloud.google.com/home/dashboard) and log in.
+ Now you can either select *New Project* at the top left of the screen or just [click here](https://console.cloud.google.com/projectcreate)
+ Create a new project, name it as you like and select a billing account. But don't worry – the Google Calendar API is free.

![new-project](../assets/gcloud-new-project.png)

+ Now you should be able to select the project in the top left corner. Most likely the project will already be selected now.
+ Either follow the [enable calendar API](https://console.cloud.google.com/flows/enableapi?apiid=calendar-json.googleapis.com) link or follow the steps below.
    + In the search bar, look for `google calendar` and select the **Google Calendar API**
    + You'll be redirected to the API description. Hit the *Enable* button.
+ Once the API is enabled, you'll be redirected again to the API management overview.
+ Before you are allowed to create the client, you will have to configure the *OAuth consent screen*
  + Click on *OAuth consent screen* on the left side (under *APIs & Services*)
  + Enter an app name, e.g. *CalendarSync* and your email address. As the user type, select *Internal*.
+ Click on *Credentials* in the sidebar and then on *Create Credentials*
+ We will need to create an *OAuth client ID*
  + Select *Desktop app* as application type and give it a name of your liking (*calendarsync-dev* maybe?)
+ Once The client ID is created you will see it in the overview.
+ Click on the download icon of the created client
    + You'll see a popup open. Copy the *Client ID* and *Client secret* into your `sync.yaml` as shown below.

![client-id-popup](../assets/gcloud-oauth-client.png)

```yaml
sink:
  adapter:
    type: google
    calendar: "jerrymccoopface@example.com"
    oAuth:
      clientId: "<clientID>"
      clientKey: "<clientSecret>"
```

If you want to use the created OAuth Application also with accounts outside of your Google Workspace, make sure to set the Usertype to `external` in the `OAuth Consent Screen` Menu.

## Apple CalDAV Adapter Setup

The Apple CalDAV adapter allows synchronization with iCloud calendars using the industry-standard CalDAV protocol. This adapter works with any Apple ID that has iCloud calendar access enabled.

### Prerequisites

Before setting up the adapter, you need:
- An Apple ID with iCloud enabled
- Access to Apple ID account settings (appleid.apple.com)
- The calendar identifier from your Calendar.app

### Setting up App-Specific Password

Since Apple requires two-factor authentication for iCloud access, you cannot use your regular Apple ID password. Instead, you must generate an app-specific password:

1. Sign in to [appleid.apple.com](https://appleid.apple.com)
2. Navigate to "Sign-In and Security"
3. Select "App-Specific Passwords"
4. Click "Generate an app-specific password"
5. Enter a label like "CalendarSync"
6. Copy the generated password (format: `xxxx-xxxx-xxxx-xxxx`)

**Important**: Save this password immediately — Apple will only show it once.

### Finding Your Calendar ID

The calendar ID is needed to specify which iCloud calendar to sync:

1. Open Calendar.app on macOS
2. Right-click on the desired calendar in the sidebar
3. Select "Get Info"
4. The calendar ID is typically the calendar name in lowercase with special characters removed
5. Common examples: `home`, `work`, `personal`

Alternatively, you can use CalDAV discovery tools or inspect the CalDAV URLs in Calendar.app's account settings.

### Configuration

```yaml
source:
  adapter:
    type: "apple"
    calendar: "personal"  # Your calendar identifier
    config: {}
    oAuth:
      clientId: "your-apple-id@icloud.com"    # Your Apple ID email
      clientKey: "xxxx-xxxx-xxxx-xxxx"        # App-specific password
      tenantId: ""                            # Leave empty for Apple CalDAV
```