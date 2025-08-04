import React from "react";
import { Link, useLocation } from "react-router-dom";
import { useMutation } from "@apollo/client";
import { Button } from "./ui/Button";
import { useAuth } from "../contexts/AuthContext";
import { LOGOUT } from "../graphql/mutations";
import { cn } from "../lib/utils";

export function Navigation() {
  const location = useLocation();
  const { user, logout: authLogout } = useAuth();
  const [logout] = useMutation(LOGOUT);

  const handleLogout = async () => {
    try {
      await logout();
      authLogout(); // Clear the auth context
    } catch (error) {
      console.error("Logout error:", error);
      // Fallback to clearing auth context even if the mutation fails
      authLogout();
    }
  };

  const navItems = [
    { path: "/dashboard", label: "Dashboard" },
    { path: "/collections", label: "Collections" },
    { path: "/wishlist", label: "Wish List" },
  ];

  return (
    <nav className="border-b bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60">
      <div className="container mx-auto px-4">
        <div className="flex h-16 items-center justify-between">
          <div className="flex items-center space-x-8">
            <Link to="/dashboard" className="flex items-center space-x-2">
              <div className="w-8 h-8 bg-gradient-to-br from-destiny-exotic to-destiny-legendary rounded-lg flex items-center justify-center">
                <span className="text-sm font-bold text-white">GT</span>
              </div>
              <span className="font-bold text-lg">Guardian Tracker</span>
            </Link>

            <div className="hidden md:flex items-center space-x-6">
              {navItems.map((item) => (
                <Link
                  key={item.path}
                  to={item.path}
                  className={cn(
                    "text-sm font-medium transition-colors hover:text-primary",
                    location.pathname === item.path
                      ? "text-primary"
                      : "text-muted-foreground"
                  )}
                >
                  {item.label}
                </Link>
              ))}
            </div>
          </div>

          <div className="flex items-center space-x-4">
            {user && (
              <div className="hidden sm:flex items-center space-x-3">
                <img
                  src="/default-avatar.png"
                  alt={user.displayName}
                  className="w-8 h-8 rounded-full"
                />
                <span className="text-sm font-medium">{user.displayName}</span>
              </div>
            )}

            <Button
              variant="ghost"
              size="sm"
              onClick={handleLogout}
              className="text-muted-foreground hover:text-foreground"
            >
              Sign Out
            </Button>
          </div>
        </div>

        {/* Mobile Navigation */}
        <div className="md:hidden border-t">
          <div className="flex items-center justify-around py-2">
            {navItems.map((item) => (
              <Link
                key={item.path}
                to={item.path}
                className={cn(
                  "text-xs font-medium px-3 py-2 rounded-md transition-colors",
                  location.pathname === item.path
                    ? "bg-primary text-primary-foreground"
                    : "text-muted-foreground hover:text-foreground hover:bg-muted"
                )}
              >
                {item.label}
              </Link>
            ))}
          </div>
        </div>
      </div>
    </nav>
  );
}
