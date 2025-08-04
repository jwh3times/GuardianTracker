import { Request, Response } from "express";
import jwt from "jsonwebtoken";

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
      const decoded = jwt.verify(token, process.env.JWT_SECRET || "dev_secret");
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
