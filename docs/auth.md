# Configuring user authentication

The server supports a few different user authentication options. This document
helps you choose and configure the option that best fits your needs:

 * **Google Single Sign On** - Configure server to authenticate accounts from
   a GSuite domain. This option is best for a server with a connection to the
   internet (Google) when your team uses GSuite identities.

 * **GitHub Sign On** - Configure server to authenticate accounts from
   one or more GitHub organizations. This option is best for a
   server with a connection to the internet (GitHub) when your team
   uses one or more GitHub organizations.

   **NOTE** - In order to prove a user is part of an organization, access must
   be granted to one of the server's configured GitHub organizations during
   the SSO login procedure.

 * **Local users** - If your server has no internet connection or you don't
   use GitHub or Google, then you can also configure the server with locally
   managed users. This mode assumes no internet access, so advanced features
   like password reset (email) and MFA (via SMS) aren't available.

## Configuring Google SSO
Assuming your satellite server will be hosted at `dg.example.com`. First go
to the [GCP Oauth2 Clients](https://console.cloud.google.com/auth/clients) 
page. From here, you'll click on "Create client". You'll be prompted for
the "Application type". Select `Web application` from the drop-down menu.
Next, give it a name like "Foundries Satellite Server".

Set the "Authorized JavaScript Origins" to a single entry. For our example, 
`https://dg.example.com`.

Set the "Authorized redirect URIs" to a single entry. For our example,
`https://dg.example.com/auth/callback`. **NOTE** - the `auth/callback` part
of the URI is critical and must be this value.

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


## Configuring locally managed users
If you can't use an SSO provider, you can configure the server with locally
managed users.

Copy `contrib/auth-config-local.json` to `<configdir>/auth/auth-config.json`
and set these optional values:

 * `Config.MaxLoginAttempts` - Set this to lock a user account after failed login attampts. For example, `5` would lockout a user account after 5 failed login attempts. The default is 0, not enforced. This setting is used in conjunction with `Config.LockoutDurationMinutes`.
 * `Config.LockoutDurationMinutes` - Set this to block a user account after failed login attempts. For example, `30` would prevent a user from loging in for 30 minutes after `Config.MaxLoginAttempts` was violated. The default is 0, not enforced.
 * `Config.MinPasswordLength` - Set this to enforce a minimum password length. For example, `8` would require passwords be at least 8 characters. The default is 0, not enforced.
 * `Config.PasswordAgeDays` - Set this to require users to change their password every `PasswordAgeDays`. For example, `180` would require a user to change their password every 180 days. The default is 0, not enforced.
 * `Config.PasswordHistory` - Set this to prevent users from repeating old passwords. For example, `5` means they must use 5 different passwords before repeating one. The default is 0, not enforced.
 * `Config.PasswordComplexityRules` - Set these options to require more complex passwords. This is disabled by default.
   * `RequireUppercase` - If true, the password must contain a character `A-Z`.
   * `RequireLowercase` - If true, the password must contain a character `a-z`.
   * `RequireDigit` - If true, the password must contain a character `0-9`.
   * `RequireSpecialChar` - If set the password must contain one of the characters in the string. For example, a value of `!@#` would make the user include one of those characters in their password.

You'll need to define the initial user by running:
```
  ./dg-sat user-add --username <initial user name> --password <password>
```