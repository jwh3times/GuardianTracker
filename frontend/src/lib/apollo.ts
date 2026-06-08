import {
  ApolloClient,
  InMemoryCache,
  HttpLink,
  ApolloLink,
  CombinedGraphQLErrors,
  ServerError,
} from "@apollo/client";
import { SetContextLink } from "@apollo/client/link/context";
import { ErrorLink } from "@apollo/client/link/error";
import { from as observableFrom } from "rxjs";
import { mergeMap } from "rxjs/operators";

// Get configuration from environment
const GRAPHQL_URL =
  import.meta.env.VITE_GRAPHQL_URL || "http://localhost:4000/graphql";
const AUTH_SERVICE_URL =
  import.meta.env.VITE_AUTH_SERVICE_URL || "http://localhost:8081";

if (import.meta.env.DEV) {
  console.log("Apollo Client Configuration:");
  console.log("  GraphQL URL:", GRAPHQL_URL);
  console.log("  Auth Service URL:", AUTH_SERVICE_URL);
}

// Token refresh helper
let isRefreshing = false;
let pendingRequests: Array<() => void> = [];

const refreshAccessToken = async (): Promise<string | null> => {
  const refreshToken = localStorage.getItem("guardian_refresh_token");

  if (!refreshToken) {
    console.error("No refresh token available");
    return null;
  }

  try {
    const response = await fetch(`${AUTH_SERVICE_URL}/api/auth/refresh`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({ refreshToken }),
    });

    if (!response.ok) {
      console.error("Token refresh failed");
      return null;
    }

    const data = await response.json();

    // Update stored tokens
    localStorage.setItem("guardian_token", data.token);
    localStorage.setItem("guardian_refresh_token", data.refreshToken);
    localStorage.setItem("guardian_user", JSON.stringify(data.user));

    console.log("Successfully refreshed access token");
    return data.token;
  } catch (error) {
    console.error("Error refreshing token:", error);
    return null;
  }
};

const resolvePendingRequests = () => {
  pendingRequests.forEach((callback) => callback());
  pendingRequests = [];
};

// HTTP Link to GraphQL endpoint
const httpLink = new HttpLink({
  uri: GRAPHQL_URL,
});

// Auth Link - adds authentication headers
const authLink = new SetContextLink((prevContext) => {
  // Get the authentication token from local storage if it exists
  const token = localStorage.getItem("guardian_token");

  // Return the headers to the context so httpLink can read them
  return {
    headers: {
      ...prevContext.headers,
      authorization: token ? `Bearer ${token}` : "",
    },
  };
});

// Clears stored auth state and bounces the user to the login screen.
const clearAuthAndRedirect = () => {
  localStorage.removeItem("guardian_token");
  localStorage.removeItem("guardian_refresh_token");
  localStorage.removeItem("guardian_user");
  window.location.href = "/login";
};

// Refreshes the access token (once), retrying the failed operation with the new
// token. Concurrent failures while a refresh is in flight queue up and retry
// once the refresh resolves.
const refreshAndRetry = (operation: ApolloLink.Operation, forward: ApolloLink.ForwardFunction) => {
  if (!isRefreshing) {
    isRefreshing = true;

    return observableFrom(
      refreshAccessToken()
        .then((newToken) => {
          if (!newToken) {
            clearAuthAndRedirect();
            return null;
          }

          // Update the operation with new token
          const oldHeaders = operation.getContext().headers;
          operation.setContext({
            headers: {
              ...oldHeaders,
              authorization: `Bearer ${newToken}`,
            },
          });

          // Release any requests that queued while the refresh was in flight
          resolvePendingRequests();
          return newToken;
        })
        .catch(() => {
          clearAuthAndRedirect();
          return null;
        })
        .finally(() => {
          isRefreshing = false;
        })
    ).pipe(mergeMap(() => forward(operation)));
  }

  // A refresh is already in flight - wait for it to complete, then retry.
  return observableFrom(
    new Promise<void>((resolve) => {
      pendingRequests.push(() => resolve());
    })
  ).pipe(mergeMap(() => forward(operation)));
};

// Error Link - handles GraphQL and network errors with token refresh
const errorLink = new ErrorLink(({ error, operation, forward }) => {
  if (CombinedGraphQLErrors.is(error)) {
    const isAuthError = error.errors.some(({ message }) =>
      ["Authentication required", "Invalid token", "Unauthorized"].some((m) =>
        message.includes(m)
      )
    );

    if (isAuthError) {
      return refreshAndRetry(operation, forward);
    }

    error.errors.forEach(({ message }) =>
      console.error(`GraphQL error: ${message}`)
    );
    return;
  }

  // Handle network 401 errors
  if (ServerError.is(error) && error.statusCode === 401) {
    return refreshAndRetry(operation, forward);
  }

  console.error(`Network error: ${error.message}`);
});

// Create Apollo Client
export const apolloClient = new ApolloClient({
  link: ApolloLink.from([errorLink, authLink, httpLink]),
  cache: new InMemoryCache({
    typePolicies: {
      User: {
        fields: {
          wishList: {
            merge(existing = [], incoming) {
              return incoming;
            },
          },
        },
      },
    },
  }),
});

export default apolloClient;
