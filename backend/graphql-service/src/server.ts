import { ApolloServer } from "@apollo/server";
import { expressMiddleware } from "@apollo/server/express4";
import { ApolloServerPluginDrainHttpServer } from "@apollo/server/plugin/drainHttpServer";
import express from "express";
import http from "http";
import cors from "cors";
import helmet from "helmet";
import morgan from "morgan";
import { typeDefs } from "./schema";
import { resolvers } from "./resolvers";
import { createContext } from "./context";

async function startServer() {
  const app = express();
  const httpServer = http.createServer(app);

  // Log configuration
  console.log("Starting GraphQL Server...");
  console.log("Environment:", process.env.NODE_ENV || "development");
  console.log("Port:", process.env.PORT || 4000);

  // Security middleware
  app.use(
    helmet({
      contentSecurityPolicy: process.env.NODE_ENV === "production", // Enable in production
    })
  );

  // Logging middleware
  app.use(morgan("combined"));

  // Apollo Server setup
  const server = new ApolloServer({
    typeDefs,
    resolvers,
    plugins: [ApolloServerPluginDrainHttpServer({ httpServer })],
    introspection: process.env.NODE_ENV !== "production",
  });

  await server.start();

  // CORS configuration
  const allowedOrigins = process.env.CORS_ALLOWED_ORIGINS
    ? process.env.CORS_ALLOWED_ORIGINS.split(",")
    : ["http://localhost:3000"];

  const corsOptions = {
    origin:
      process.env.NODE_ENV === "production"
        ? allowedOrigins
        : ["http://localhost:3000"],
    credentials: true,
  };

  console.log("CORS allowed origins:", corsOptions.origin);

  // GraphQL endpoint
  app.use(
    "/graphql",
    cors<cors.CorsRequest>(corsOptions),
    express.json(),
    expressMiddleware(server, {
      context: createContext,
    })
  );

  // Health check endpoint
  app.get("/health", (_, res) => {
    res.json({ status: "ok", timestamp: new Date().toISOString() });
  });

  const port = process.env.PORT || 4000;

  await new Promise<void>((resolve) => httpServer.listen({ port }, resolve));

  console.log(`🚀 GraphQL Server ready at http://localhost:${port}/graphql`);

  if (process.env.NODE_ENV !== "production") {
    console.log(
      `📊 GraphQL Playground available at http://localhost:${port}/graphql`
    );
  }
}

startServer().catch((error) => {
  console.error("Error starting server:", error);
  process.exit(1);
});
