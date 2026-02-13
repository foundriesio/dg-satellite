# Configuring User Authentication

The server supports a few different user authentication options. This document
helps you choose and configure the option that best fits your needs:

* **Google Single Sign On** — Configure server to authenticate accounts from
   a GSuite domain. This option is best for a server with a connection to the
   internet (Google) when your team uses GSuite identities.

* **GitHub Sign On** — Configure server to authenticate accounts from
    one or more GitHub organizations. This option is best for a
   server with a connection to the internet (GitHub) when your team
   uses one or more GitHub organizations.

   **NOTE** — In order to prove a user is part of an organization, access must
   be granted to one of the server's configured GitHub organizations during
   the SSO login procedure.

* **Local users** — If your server has no internet connection or you do not
   use GitHub or Google, you can also configure the server with locally
   managed users. This mode assumes no internet access, so advanced features
   like password reset (email) and MFA (via SMS) are not available.

## Configuring Google SSO

Assuming your satellite server will be hosted at `dg.example.com`. First go
to the [GCP Oauth2 Clients](https://console.cloud.google.com/auth/clients)
page. From here, you'll click on "Create client". You'll be prompted for
the "Application type". Select `Web application` from the drop-down menu.
Next, give it a name like "Foundries Satellite Server".

Set the "Authorized JavaScript Origins" to a single entry. For our example,
`https://dg.example.com`.

Set the "Authorized redirect URIs" to a single entry. For our example,
`https://dg.example.com/auth/callback`.

> [!IMPORTANT]
> the `auth/callback` part of the URI is critical and must be this value.

After clicking "Create", you'll be presented with a pop-up dialog that includes
your Client ID and Secret. Make note of both these values. They are required
for the next step.

Copy `/contrib/auth-config-google.json` to `<configdir>/auth/auth-config.json`
and set the values:

* `Config.ClientID`
* `Config.ClientSecret`
* `Config.AllowedDomains` - e.g. If your company emails are `@example.com` - enter `example.com` here.
* `Config.BaseUrl` - For our example, `https://dg.example.com`.

## Configuring GitHub SSO

Assuming your satellite server will be hosted at `dg.example.com`. First go
to the GitHub [Developer Settings](https://github.com/settings/apps) page.
From here, select the "OAuth Apps" option on the side and then click the
"New OAuth App" button. The "Application name" should be something descriptive
for you like "Foundries Satellite Server". The URL does not matter, but could
be `https://dg.example.com` for this example. The "Authorization callback URL"
is critical and must be `https://dg.example.com/auth/callback`. You can
then click "Register application". This will take you to a page where you
can manage this new application. The "Client ID" will be displayed in plain
text. You'll also need to generate a client secret by clicking "Generate a new
client secret". These two values are required for the next step.

Copy `/contrib/auth-config-github.json` to `<configdir>/auth/auth-config.json`
and set the values:

* `Config.ClientID`
* `Config.ClientSecret`
* `Config.AllowedOrgs` - A user must be a member of one of the values here to login to the server.
* `Config.BaseUrl` - For our example, `https://dg.example.com`.

## Configuring Locally Managed Users

TODO
