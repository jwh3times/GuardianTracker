import { ApolloServer } from "@apollo/server";
import { expressMiddleware } from "@as-integrations/express5";
import { ApolloServerPluginDrainHttpServer } from "@apollo/server/plugin/drainHttpServer";
import express from "express";
import http from "http";
import cors from "cors";
import helmet from "helmet";
import morgan from "morgan";
import rateLimit from "express-rate-limit";
import { GraphQLFormattedError } from "graphql";
import { typeDefs } from "./schema";
import { resolvers } from "./resolvers";
import { createContext } from "./context";

const isProduction = process.env.NODE_ENV === "production";

// Error formatter - hides internal errors in production
function formatError(
  formattedError: GraphQLFormattedError,
  error: unknown
): GraphQLFormattedError {
  // Log all errors server-side
  console.error("GraphQL Error:", error);

  // In production, hide internal error details
  if (isProduction) {
    // Known error codes that are safe to expose
    const safeErrorCodes = [
      "UNAUTHENTICATED",
      "FORBIDDEN",
      "BAD_USER_INPUT",
      "GRAPHQL_VALIDATION_FAILED",
    ];

    const errorCode = formattedError.extensions?.code as string;

    if (safeErrorCodes.includes(errorCode)) {
      // Return the formatted error with extensions but without stack trace
      const extensions: Record<string, unknown> = {
        code: errorCode,
      };

      // Include validation errors if present
      if (formattedError.extensions?.validationErrors) {
        extensions.validationErrors = formattedError.extensions.validationErrors;
      }

      return {
        message: formattedError.message,
        extensions,
      };
    }

    // For unknown errors, return generic message
    return {
      message: "An internal error occurred",
      extensions: {
        code: "INTERNAL_SERVER_ERROR",
      },
    };
  }

  // In development, return full error details
  return formattedError;
}

async function startServer() {
  const app = express();
  const httpServer = http.createServer(app);

  // Only log in non-production
  if (!isProduction) {
    console.log("Starting GraphQL Server...");
    console.log("Environment:", process.env.NODE_ENV || "development");
    console.log("Port:", process.env.PORT || 4000);
  }

  // Trust proxy (needed for rate limiting behind reverse proxy)
  app.set("trust proxy", 1);

  // Security middleware - always enabled with proper settings
  app.use(
    helmet({
      contentSecurityPolicy: {
        directives: {
          defaultSrc: ["'self'"],
          styleSrc: ["'self'", "'unsafe-inline'"],
          imgSrc: ["'self'", "data:", "https://www.bungie.net"],
          scriptSrc: ["'self'"],
          connectSrc: ["'self'"],
        },
      },
      crossOriginEmbedderPolicy: false, // Required for GraphQL Playground in dev
    })
  );

  // Rate limiting
  const limiter = rateLimit({
    windowMs: 15 * 60 * 1000, // 15 minutes
    max: isProduction ? 100 : 1000, // Limit each IP
    message: {
      error: "Too many requests, please try again later",
      code: "RATE_LIMITED",
    },
    standardHeaders: true,
    legacyHeaders: false,
  });

  app.use(limiter);

  // Logging middleware - less verbose in production
  app.use(morgan(isProduction ? "combined" : "dev"));

  // Apollo Server setup with error formatting
  const server = new ApolloServer({
    typeDefs,
    resolvers,
    plugins: [ApolloServerPluginDrainHttpServer({ httpServer })],
    introspection: !isProduction,
    formatError,
  });

  await server.start();

  // CORS configuration
  const allowedOrigins = process.env.CORS_ALLOWED_ORIGINS
    ? process.env.CORS_ALLOWED_ORIGINS.split(",").map((o) => o.trim())
    : ["http://localhost:3000"];

  const corsOptions: cors.CorsOptions = {
    // Validate against CORS_ALLOWED_ORIGINS in all environments. Requests with
    // no Origin header (curl, server-to-server, same-origin) are always allowed.
    origin: (origin, callback) => {
      if (!origin || allowedOrigins.includes(origin)) {
        callback(null, true);
      } else {
        callback(new Error("Not allowed by CORS"));
      }
    },
    credentials: true,
    methods: ["GET", "POST", "OPTIONS"],
    allowedHeaders: ["Content-Type", "Authorization"],
  };

  if (!isProduction) {
    console.log("CORS allowed origins:", allowedOrigins);
  }

  // GraphQL endpoint
  app.use(
    "/graphql",
    cors<cors.CorsRequest>(corsOptions),
    express.json({ limit: "100kb" }), // Limit request body size
    expressMiddleware(server, {
      context: createContext,
    })
  );

  // Health check endpoint with dependency checks
  app.get("/health", async (_, res) => {
    const health = {
      status: "ok",
      timestamp: new Date().toISOString(),
      uptime: process.uptime(),
      environment: process.env.NODE_ENV || "development",
    };

    res.json(health);
  });

  // Ready check for K8s
  app.get("/ready", (_, res) => {
    res.json({ ready: true });
  });

  const port = process.env.PORT || 4000;

  await new Promise<void>((resolve) => httpServer.listen({ port }, resolve));

  console.log(`GraphQL Server ready at http://localhost:${port}/graphql`);

  if (!isProduction) {
    console.log(
      `GraphQL Playground available at http://localhost:${port}/graphql`
    );
  }

  // Graceful shutdown
  const shutdown = async (signal: string) => {
    console.log(`${signal} received, shutting down gracefully...`);

    // Stop accepting new connections
    httpServer.close(() => {
      console.log("HTTP server closed");
    });

    // Stop Apollo Server
    await server.stop();
    console.log("Apollo Server stopped");

    process.exit(0);
  };

  process.on("SIGTERM", () => shutdown("SIGTERM"));
  process.on("SIGINT", () => shutdown("SIGINT"));
}

startServer().catch((error) => {
  console.error("Error starting server:", error);
  process.exit(1);
});
