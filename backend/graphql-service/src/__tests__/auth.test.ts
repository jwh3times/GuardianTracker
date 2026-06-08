import { GraphQLError } from 'graphql';
import { requireAuth, getAuthenticatedUser, requireOwnership } from '../utils/auth.js';
import { Context, AuthUser } from '../context.js';
import { Request, Response } from 'express';

// Mock request and response
const mockReq = {} as Request;
const mockRes = {} as Response;

// Factory for creating test users
const createMockUser = (overrides: Partial<AuthUser> = {}): AuthUser => ({
  membership_id: '12345678901234567890',
  membershipId: '12345678901234567890',
  display_name: 'TestGuardian',
  displayName: 'TestGuardian',
  membership_type: 3,
  membershipType: 3,
  platform: 'Steam',
  token_type: 'access',
  iat: Math.floor(Date.now() / 1000),
  exp: Math.floor(Date.now() / 1000) + 3600,
  ...overrides,
});

// Factory for creating test contexts
const createMockContext = (user: AuthUser | null = null): Context => ({
  user,
  req: mockReq,
  res: mockRes,
});

describe('Auth Utilities', () => {
  describe('requireAuth', () => {
    it('should not throw when user is authenticated', () => {
      const context = createMockContext(createMockUser());
      expect(() => requireAuth(context)).not.toThrow();
    });

    it('should throw UNAUTHENTICATED error when user is null', () => {
      const context = createMockContext(null);

      expect(() => requireAuth(context)).toThrow(GraphQLError);

      try {
        requireAuth(context);
      } catch (error) {
        expect(error).toBeInstanceOf(GraphQLError);
        const gqlError = error as GraphQLError;
        expect(gqlError.message).toBe('Authentication required');
        expect(gqlError.extensions?.code).toBe('UNAUTHENTICATED');
        expect(gqlError.extensions?.http).toEqual({ status: 401 });
      }
    });
  });

  describe('getAuthenticatedUser', () => {
    it('should return user when authenticated', () => {
      const mockUser = createMockUser();
      const context = createMockContext(mockUser);

      const result = getAuthenticatedUser(context);

      expect(result).toBe(mockUser);
    });

    it('should throw when user is not authenticated', () => {
      const context = createMockContext(null);

      expect(() => getAuthenticatedUser(context)).toThrow(GraphQLError);
    });
  });

  describe('requireOwnership', () => {
    it('should not throw when user owns the resource (membershipId match)', () => {
      const mockUser = createMockUser({ membershipId: 'owner123456789012' });
      const context = createMockContext(mockUser);

      expect(() => requireOwnership(context, 'owner123456789012')).not.toThrow();
    });

    it('should not throw when user owns the resource (membership_id match)', () => {
      const mockUser = createMockUser({ membership_id: 'owner123456789012' });
      const context = createMockContext(mockUser);

      expect(() => requireOwnership(context, 'owner123456789012')).not.toThrow();
    });

    it('should throw FORBIDDEN when user does not own resource', () => {
      const mockUser = createMockUser({
        membershipId: 'user1234567890123',
        membership_id: 'user1234567890123',
      });
      const context = createMockContext(mockUser);

      expect(() => requireOwnership(context, 'other_owner_id12345')).toThrow(GraphQLError);

      try {
        requireOwnership(context, 'other_owner_id12345');
      } catch (error) {
        expect(error).toBeInstanceOf(GraphQLError);
        const gqlError = error as GraphQLError;
        expect(gqlError.message).toBe("You don't have permission to access this resource");
        expect(gqlError.extensions?.code).toBe('FORBIDDEN');
        expect(gqlError.extensions?.http).toEqual({ status: 403 });
      }
    });

    it('should throw UNAUTHENTICATED when user is not logged in', () => {
      const context = createMockContext(null);

      expect(() => requireOwnership(context, 'any_owner_id')).toThrow(GraphQLError);

      try {
        requireOwnership(context, 'any_owner_id');
      } catch (error) {
        const gqlError = error as GraphQLError;
        expect(gqlError.extensions?.code).toBe('UNAUTHENTICATED');
      }
    });
  });
});
