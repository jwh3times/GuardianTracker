import {
  ApolloClient,
  InMemoryCache,
  createHttpLink,
  from,
} from "@apollo/client";
import { setContext } from "@apollo/client/link/context";
import { onError } from "@apollo/client/link/error";

// HTTP Link to GraphQL endpoint
const httpLink = createHttpLink({
  uri: "http://localhost:4000/graphql",
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

// Error Link - handles GraphQL and network errors
const errorLink = onError(
  ({ graphQLErrors, networkError, operation, forward }) => {
    if (graphQLErrors) {
      graphQLErrors.forEach(({ message, locations, path }) => {
        console.error(
          `GraphQL error: Message: ${message}, Location: ${locations}, Path: ${path}`
        );

        // Handle authentication errors
        if (
          message.includes("Authentication required") ||
          message.includes("Invalid token")
        ) {
          // Clear stored authentication data
          localStorage.removeItem("guardian_token");
          localStorage.removeItem("guardian_user");

          // Redirect to login page
          window.location.href = "/login";
        }
      });
    }

    if (networkError) {
      console.error(`Network error: ${networkError}`);

      // Handle network errors (e.g., server down)
      if ("statusCode" in networkError && networkError.statusCode === 401) {
        // Unauthorized - clear auth and redirect
        localStorage.removeItem("guardian_token");
        localStorage.removeItem("guardian_user");
        window.location.href = "/login";
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
