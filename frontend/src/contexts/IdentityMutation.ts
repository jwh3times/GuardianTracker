import { createContext, useContext } from "react";
import {
  useMutation,
  type DefaultError,
  type UseMutationOptions,
} from "@tanstack/react-query";
import { ApiError } from "../lib/api";
import type { ApplicationIdentity } from "../lib/applicationIdentity";

export const IdentityContext = createContext<ApplicationIdentity | undefined>(
  undefined,
);

/** Fence work delayed by optimistic preparation, retries, or a departed provider. */
export function useIdentityMutation<
  TData = unknown,
  TError = DefaultError,
  TVariables = void,
  TContext = unknown,
>(options: UseMutationOptions<TData, TError, TVariables, TContext>) {
  const scope = useContext(IdentityContext);
  if (!scope)
    throw new Error("useIdentityMutation must be used within AppProviders");
  const assertCurrent = () => {
    if (!scope.isCurrent())
      throw new ApiError("The browser session changed", 401, "SESSION_CHANGED");
  };
  return useMutation<TData, TError, TVariables, TContext>({
    ...options,
    mutationFn: options.mutationFn
      ? (...args) => {
          assertCurrent();
          return options.mutationFn!(...args);
        }
      : undefined,
    onMutate: options.onMutate
      ? (...args) => {
          assertCurrent();
          return options.onMutate!(...args);
        }
      : undefined,
    onSuccess: (...args) =>
      scope.isCurrent() ? options.onSuccess?.(...args) : undefined,
    onError: (...args) =>
      scope.isCurrent() ? options.onError?.(...args) : undefined,
    onSettled: (...args) =>
      scope.isCurrent() ? options.onSettled?.(...args) : undefined,
  });
}
