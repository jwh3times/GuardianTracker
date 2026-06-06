import {
  ApolloClient,
  InMemoryCache,
  createHttpLink,
  from,
  fromPromise,
} from "@apollo/client";
import { setContext } from "@apollo/client/link/context";
import { onError } from "@apollo/client/link/error";

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
const httpLink = createHttpLink({
  uri: GRAPHQL_URL,
});

// Auth Link - adds authentication headers
const authLink = setContext((_, { headers }) => {
  // Get the authentication token from local storage if it exists
  const token = localStorage.getItem("guardian_token");

  // Return the headers to the context so httpLink can read them
  return {
    headers: {
      ...headers,
      authorization: token ? `Bearer ${token}` : "",
    },
  };
});

// Error Link - handles GraphQL and network errors with token refresh
const errorLink = onError(
  ({ graphQLErrors, networkError, operation, forward }) => {
    if (graphQLErrors) {
      for (const err of graphQLErrors) {
        const { message } = err;
        console.error(`GraphQL error: ${message}`);

        // Handle authentication errors - try to refresh token
        if (
          message.includes("Authentication required") ||
          message.includes("Invalid token") ||
          message.includes("Unauthorized")
        ) {
          if (!isRefreshing) {
            isRefreshing = true;

            return fromPromise(
              refreshAccessToken()
                .then((newToken) => {
                  if (!newToken) {
                    // Refresh failed, logout
                    localStorage.removeItem("guardian_token");
                    localStorage.removeItem("guardian_refresh_token");
                    localStorage.removeItem("guardian_user");
                    window.location.href = "/login";
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

                  // Retry the request
                  resolvePendingRequests();
                  return newToken;
                })
                .catch(() => {
                  // Refresh failed
                  localStorage.removeItem("guardian_token");
                  localStorage.removeItem("guardian_refresh_token");
                  localStorage.removeItem("guardian_user");
                  window.location.href = "/login";
                  return null;
                })
                .finally(() => {
                  isRefreshing = false;
                })
            ).flatMap(() => forward(operation));
          } else {
            // Wait for refresh to complete
            return fromPromise(
              new Promise<void>((resolve) => {
                pendingRequests.push(() => resolve());
              })
            ).flatMap(() => forward(operation));
          }
        }
      }
    }

    if (networkError) {
      console.error(`Network error: ${networkError}`);

      // Handle network 401 errors
      if ("statusCode" in networkError && networkError.statusCode === 401) {
        if (!isRefreshing) {
          isRefreshing = true;

          return fromPromise(
            refreshAccessToken()
              .then((newToken) => {
                if (!newToken) {
                  localStorage.removeItem("guardian_token");
                  localStorage.removeItem("guardian_refresh_token");
                  localStorage.removeItem("guardian_user");
                  window.location.href = "/login";
                  return null;
                }

                const oldHeaders = operation.getContext().headers;
                operation.setContext({
                  headers: {
                    ...oldHeaders,
                    authorization: `Bearer ${newToken}`,
                  },
                });

                resolvePendingRequests();
                return newToken;
              })
              .catch(() => {
                localStorage.removeItem("guardian_token");
                localStorage.removeItem("guardian_refresh_token");
                localStorage.removeItem("guardian_user");
                window.location.href = "/login";
                return null;
              })
              .finally(() => {
                isRefreshing = false;
              })
          ).flatMap(() => forward(operation));
        }
      }
    }
  }
);

// Create Apollo Client
export const apolloClient = new ApolloClient({
  link: from([errorLink, authLink, httpLink]),
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
  defaultOptions: {
    watchQuery: {
      errorPolicy: "all",
    },
    query: {
      errorPolicy: "all",
    },
  },
});

export default apolloClient;
