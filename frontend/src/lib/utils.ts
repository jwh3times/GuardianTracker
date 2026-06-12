import { type ClassValue, clsx } from "clsx";
import { twMerge } from "tailwind-merge";

/** Merge class names, resolving Tailwind conflicts (used by legacy ui/ components). */
export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}
