import { Request, Response } from "express";
import jwt from "jsonwebtoken";

const JWT_SECRET = process.env.JWT_SECRET || "dev_secret";

// Warn if using default secret
if (JWT_SECRET === "dev_secret" && process.env.NODE_ENV === "production") {
  console.error("CRITICAL: Using default JWT_SECRET in production!");
  throw new Error("JWT_SECRET must be set in production environment");
}

export interface Context {
  user?: any;
  req: Request;
  res: Response;
}

export async function createContext({
  req,
  res,
}: {
  req: Request;
  res: Response;
}): Promise<Context> {
  // Extract user from JWT token
  let user = null;
  const authHeader = req.headers.authorization;

  if (authHeader) {
    const token = authHeader.replace("Bearer ", "");
    try {
      const decoded = jwt.verify(token, JWT_SECRET);
      user = decoded;
    } catch (error) {
      console.warn("Invalid JWT token:", error);
    }
  }

  return {
    user,
    req,
    res,
  };
}
