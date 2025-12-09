import { Request, Response } from "express";
import jwt from "jsonwebtoken";

const JWT_SECRET = process.env.JWT_SECRET || "dev_secret";

// Warn if using default secret
if (JWT_SECRET === "dev_secret" && process.env.NODE_ENV === "production") {
  console.error("CRITICAL: Using default JWT_SECRET in production!");
  throw new Error("JWT_SECRET must be set in production environment");
}

// User type from JWT claims
export interface AuthUser {
  membership_id: string;
  membershipId: string;
  display_name: string;
  displayName: string;
  membership_type: number;
  membershipType: number;
  platform: string;
  token_type: string;
  iat: number;
  exp: number;
}

export interface Context {
  user: AuthUser | null;
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
  let user: AuthUser | null = null;
  const authHeader = req.headers.authorization;

  if (authHeader && authHeader.startsWith("Bearer ")) {
    const token = authHeader.slice(7); // Remove "Bearer " prefix

    try {
      const decoded = jwt.verify(token, JWT_SECRET, {
        algorithms: ["HS256"],
      }) as AuthUser;

      // Validate required claims
      if (decoded.membership_id || decoded.membershipId) {
        user = decoded;
      }
    } catch (error) {
      // Token validation failed - user remains null
      // Don't log in production to avoid noise from expired tokens
      if (process.env.NODE_ENV !== "production") {
        console.warn("JWT validation failed:", (error as Error).message);
      }
    }
  }

  return {
    user,
    req,
    res,
  };
}
