import { FormEvent, useEffect, useState } from "react";
import { Button } from "@/components/ui/button";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

const INTERNAL_PREFIX = "/__auth_proxy__";
const LOGIN_PAGE = `${INTERNAL_PREFIX}/pages/login`;
const LOGIN_API = `${INTERNAL_PREFIX}/api/login`;
const SESSION_API = `${INTERNAL_PREFIX}/api/session`;

type SessionResponse = {
  authenticated: boolean;
  username?: string;
};

function nextTarget() {
  const next = new URLSearchParams(window.location.search).get("next");
  if (next && next.startsWith("/")) {
    return next;
  }
  return "/";
}

function completeLogin() {
  if (window.location.pathname === LOGIN_PAGE) {
    window.location.assign(nextTarget());
    return;
  }
  window.location.reload();
}

export function App() {
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [checking, setChecking] = useState(true);
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    let active = true;

    async function bootstrap() {
      try {
        const response = await fetch(SESSION_API, {
          credentials: "same-origin",
          headers: {
            Accept: "application/json",
          },
        });
        const payload = (await response.json()) as SessionResponse;
        if (!active) {
          return;
        }
        if (payload.authenticated) {
          completeLogin();
          return;
        }
      } catch {
        if (active) {
          setError("Session check failed. You can still try signing in.");
        }
      } finally {
        if (active) {
          setChecking(false);
        }
      }
    }

    bootstrap();
    return () => {
      active = false;
    };
  }, []);

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setSubmitting(true);
    setError("");

    try {
      const response = await fetch(LOGIN_API, {
        method: "POST",
        credentials: "same-origin",
        headers: {
          "Content-Type": "application/json",
          Accept: "application/json",
        },
        body: JSON.stringify({ username, password }),
      });

      if (!response.ok) {
        const payload = (await response.json().catch(() => null)) as { error?: string } | null;
        setError(payload?.error ?? "Sign-in failed.");
        return;
      }

      completeLogin();
    } catch {
      setError("The proxy could not be reached. Try again.");
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <main className="flex min-h-screen items-center justify-center bg-black px-4 py-8">
      <Card className="w-full max-w-sm">
        <CardHeader>
          <CardTitle>Login to your account</CardTitle>
          <CardDescription>Enter your username below to login to your account</CardDescription>
        </CardHeader>
        <CardContent>
          <form className="grid gap-6" onSubmit={handleSubmit}>
            <div className="grid gap-2">
              <Label htmlFor="username">Username</Label>
              <Input
                id="username"
                autoComplete="username"
                disabled={checking || submitting}
                onChange={(event) => setUsername(event.target.value)}
                placeholder="operator"
                required
                value={username}
              />
            </div>

            <div className="grid gap-2">
              <Label htmlFor="password">Password</Label>
              <Input
                id="password"
                autoComplete="current-password"
                disabled={checking || submitting}
                onChange={(event) => setPassword(event.target.value)}
                required
                type="password"
                value={password}
              />
            </div>

            {error ? (
              <Alert variant="destructive">
                <AlertDescription>{error}</AlertDescription>
              </Alert>
            ) : null}

            <CardFooter className="p-0">
              <Button className="w-full" disabled={checking || submitting || !username || !password} type="submit">
                {checking ? "Checking session..." : submitting ? "Logging in..." : "Login"}
              </Button>
            </CardFooter>
          </form>
        </CardContent>
      </Card>
    </main>
  );
}
