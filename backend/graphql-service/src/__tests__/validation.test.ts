import { GraphQLError } from 'graphql';
import { schemas, validationSchemas, validateInput } from '../utils/validation';

describe('Validation Schemas', () => {
  describe('membershipId', () => {
    it('should accept valid membership IDs', () => {
      expect(schemas.membershipId.parse('12345678901234567890')).toBe('12345678901234567890');
      expect(schemas.membershipId.parse('1234567890')).toBe('1234567890');
    });

    it('should reject membership IDs that are too short', () => {
      expect(() => schemas.membershipId.parse('123456789')).toThrow();
    });

    it('should reject membership IDs that are too long', () => {
      const longId = '1'.repeat(26);
      expect(() => schemas.membershipId.parse(longId)).toThrow();
    });

    it('should reject non-numeric membership IDs', () => {
      expect(() => schemas.membershipId.parse('abc1234567890')).toThrow();
      expect(() => schemas.membershipId.parse('12345-67890')).toThrow();
    });
  });

  describe('membershipType', () => {
    it('should accept valid membership types', () => {
      expect(schemas.membershipType.parse(1)).toBe(1); // Xbox
      expect(schemas.membershipType.parse(2)).toBe(2); // PSN
      expect(schemas.membershipType.parse(3)).toBe(3); // Steam
    });

    it('should reject invalid membership types', () => {
      expect(() => schemas.membershipType.parse(0)).toThrow();
      expect(() => schemas.membershipType.parse(999)).toThrow();
    });

    it('should reject non-integer membership types', () => {
      expect(() => schemas.membershipType.parse(1.5)).toThrow();
    });
  });

  describe('itemHash', () => {
    it('should accept valid item hashes', () => {
      expect(schemas.itemHash.parse('1234567890')).toBe('1234567890');
    });

    it('should reject empty item hashes', () => {
      expect(() => schemas.itemHash.parse('')).toThrow();
    });

    it('should reject non-numeric item hashes', () => {
      expect(() => schemas.itemHash.parse('abc123')).toThrow();
    });
  });

  describe('priority', () => {
    it('should accept valid priorities', () => {
      expect(schemas.priority.parse('LOW')).toBe('LOW');
      expect(schemas.priority.parse('MEDIUM')).toBe('MEDIUM');
      expect(schemas.priority.parse('HIGH')).toBe('HIGH');
    });

    it('should accept undefined', () => {
      expect(schemas.priority.parse(undefined)).toBeUndefined();
    });

    it('should reject invalid priorities', () => {
      expect(() => schemas.priority.parse('URGENT')).toThrow();
    });
  });

  describe('notes', () => {
    it('should accept valid notes', () => {
      expect(schemas.notes.parse('This is a note')).toBe('This is a note');
    });

    it('should trim whitespace', () => {
      expect(schemas.notes.parse('  trimmed  ')).toBe('trimmed');
    });

    it('should reject notes that are too long', () => {
      const longNote = 'a'.repeat(501);
      expect(() => schemas.notes.parse(longNote)).toThrow();
    });
  });

  describe('searchQuery', () => {
    it('should accept valid search queries', () => {
      expect(schemas.searchQuery.parse('Gjallarhorn')).toBe('Gjallarhorn');
    });

    it('should trim whitespace', () => {
      expect(schemas.searchQuery.parse('  query  ')).toBe('query');
    });

    it('should reject queries that are too short', () => {
      expect(() => schemas.searchQuery.parse('a')).toThrow();
    });

    it('should reject queries that are too long', () => {
      const longQuery = 'a'.repeat(101);
      expect(() => schemas.searchQuery.parse(longQuery)).toThrow();
    });
  });
});

describe('Composite Validation Schemas', () => {
  describe('userCollections', () => {
    it('should accept valid input', () => {
      const input = {
        membershipType: 3,
        membershipId: '12345678901234567890',
      };
      expect(validationSchemas.userCollections.parse(input)).toEqual(input);
    });

    it('should reject invalid membership type', () => {
      const input = {
        membershipType: 999,
        membershipId: '12345678901234567890',
      };
      expect(() => validationSchemas.userCollections.parse(input)).toThrow();
    });
  });

  describe('addToWishList', () => {
    it('should accept valid input with all fields', () => {
      const input = {
        itemHash: '1234567890',
        priority: 'HIGH' as const,
        notes: 'Must get this',
      };
      expect(validationSchemas.addToWishList.parse(input)).toEqual({
        ...input,
        notes: 'Must get this',
      });
    });

    it('should accept input without optional fields', () => {
      const input = {
        itemHash: '1234567890',
      };
      expect(validationSchemas.addToWishList.parse(input)).toEqual(input);
    });
  });

  describe('searchItems', () => {
    it('should accept valid search input', () => {
      const input = {
        query: 'Exotic weapon',
        filters: {
          isExotic: true,
        },
      };
      expect(validationSchemas.searchItems.parse(input)).toEqual(input);
    });

    it('should accept search without filters', () => {
      const input = {
        query: 'Gjallarhorn',
      };
      expect(validationSchemas.searchItems.parse(input)).toEqual(input);
    });
  });
});

describe('validateInput', () => {
  it('should return valid data on success', () => {
    const input = {
      membershipType: 3,
      membershipId: '12345678901234567890',
    };
    const result = validateInput(validationSchemas.userCollections, input);
    expect(result).toEqual(input);
  });

  it('should throw GraphQLError on validation failure', () => {
    const input = {
      membershipType: 999,
      membershipId: 'invalid',
    };

    expect(() => validateInput(validationSchemas.userCollections, input)).toThrow(GraphQLError);
  });

  it('should include validation errors in GraphQL error extensions', () => {
    const input = {
      membershipType: 999,
      membershipId: 'invalid',
    };

    try {
      validateInput(validationSchemas.userCollections, input);
      fail('Expected error to be thrown');
    } catch (error) {
      expect(error).toBeInstanceOf(GraphQLError);
      const gqlError = error as GraphQLError;
      expect(gqlError.extensions?.code).toBe('BAD_USER_INPUT');
      expect(gqlError.extensions?.validationErrors).toBeDefined();
      expect(Array.isArray(gqlError.extensions?.validationErrors)).toBe(true);
    }
  });
});
