"use client";

import { Suspense, useEffect, useState } from "react";
import { useSearchParams, useRouter } from "next/navigation";
import {
  exchangeCodeForToken,
  validateState,
  TokenStorage,
  verifyToken,
} from "@/lib/eve-sso";

function CallbackContent() {
  const searchParams = useSearchParams();
  const router = useRouter();
  const [status, setStatus] = useState<"loading" | "success" | "error">("loading");
  const [message, setMessage] = useState("Processing authentication...");

  useEffect(() => {
    const handleCallback = async () => {
      try {
        const code = searchParams.get("code");
        const state = searchParams.get("state");

        if (!code) {
          setStatus("error");
          setMessage("No authorization code received");
          return;
        }

        if (!state) {
          setStatus("error");
          setMessage("No state parameter received");
          return;
        }

        // Validate state to prevent CSRF
        if (!validateState(state)) {
          setStatus("error");
          setMessage("Invalid state parameter - possible CSRF attack");
          return;
        }

        setMessage("Exchanging authorization code for token...");

        // Exchange code for token
        const clientId = process.env.NEXT_PUBLIC_EVE_CLIENT_ID || "0828b4bcd20242aeb9b8be10f5451094";

        const token = await exchangeCodeForToken(code, clientId);

        setMessage("Verifying character information...");

        // Verify token and get character info
        const charInfo = await verifyToken(token.access_token);

        // Save token and character info
        TokenStorage.save(token);
        TokenStorage.saveCharacterInfo(charInfo);

        setStatus("success");
        setMessage(`Successfully logged in as ${charInfo.CharacterName}! Redirecting...`);

        // Dispatch custom event to notify AuthContext
        window.dispatchEvent(new Event("eve-login-success"));

        // Redirect to home page after 1 second
        setTimeout(() => {
          router.push("/");
        }, 1000);
      } catch (error) {
        console.error("Callback error:", error);
        setStatus("error");
        setMessage(
          error instanceof Error ? error.message : "Failed to complete authentication"
        );
      }
    };

    handleCallback();
  }, [searchParams, router]);

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-50">
      <div className="max-w-md w-full space-y-8 p-8">
        <div className="text-center">
          {status === "loading" && (
            <div className="space-y-4">
              <div className="inline-block h-12 w-12 animate-spin rounded-full border-4 border-solid border-blue-600 border-r-transparent"></div>
              <h2 className="text-2xl font-bold text-gray-900">Authenticating</h2>
              <p className="text-gray-600">{message}</p>
            </div>
          )}

          {status === "success" && (
            <div className="space-y-4">
              <div className="text-6xl">✓</div>
              <h2 className="text-2xl font-bold text-green-600">Login Successful</h2>
              <p className="text-gray-600">{message}</p>
            </div>
          )}

          {status === "error" && (
            <div className="space-y-4">
              <div className="text-6xl">✗</div>
              <h2 className="text-2xl font-bold text-red-600">Login Failed</h2>
              <p className="text-gray-600">{message}</p>
              <button
                onClick={() => router.push("/")}
                className="mt-4 px-4 py-2 bg-gray-800 text-white rounded hover:bg-gray-700"
              >
                Return to Home
              </button>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

export default function CallbackPage() {
  return (
    <Suspense
      fallback={
        <div className="min-h-screen flex items-center justify-center bg-gray-50">
          <div className="text-center">
            <div className="inline-block h-12 w-12 animate-spin rounded-full border-4 border-solid border-blue-600 border-r-transparent"></div>
            <h2 className="text-2xl font-bold text-gray-900 mt-4">Loading...</h2>
          </div>
        </div>
      }
    >
      <CallbackContent />
    </Suspense>
  );
}
