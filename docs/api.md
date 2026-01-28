# Using the REST API

## Authentication

The REST API is protected by API tokens. To create a token, you must go to the
settings page in the UI and create a token.

The token must be provided using the `Authorization: Bearer <token>` header.
For example, you could list devices using cURL with:
```
 $ curl -H "Authorization: Bearer <your token>" http://localhost:8000/devices
```

## API documentation

TODO: How to see swagger docs
