"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";

export default function CallbackPage() {
  const router = useRouter();
  const [status, setStatus] = useState<"processing" | "success" | "error">("processing");
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const handleCallback = async () => {
      try {
        // The backend has already processed the OAuth callback and set the session cookie
        // We just need to verify the session and redirect
        const API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8082";
        
        const response = await fetch(`${API_URL}/api/v1/auth/verify`, {
          credentials: "include",
        });

        if (response.ok) {
          setStatus("success");
          // Redirect to home page after successful login
          setTimeout(() => {
            router.push("/");
          }, 1500);
        } else {
          setStatus("error");
          setError("Failed to verify session. Please try logging in again.");
        }
      } catch (err) {
        setStatus("error");
        setError("An unexpected error occurred. Please try again.");
        console.error("Callback error:", err);
      }
    };

    handleCallback();
  }, [router]);

  return (
    <div className="flex min-h-screen items-center justify-center">
      <div className="text-center">
        {status === "processing" && (
          <>
            <div className="mb-4 inline-block h-12 w-12 animate-spin rounded-full border-4 border-solid border-current border-r-transparent"></div>
            <h1 className="text-2xl font-bold">Processing Login...</h1>
            <p className="mt-2 text-muted-foreground">Please wait while we log you in</p>
          </>
        )}
        
        {status === "success" && (
          <>
            <div className="mb-4 text-6xl">✓</div>
            <h1 className="text-2xl font-bold text-green-600">Login Successful!</h1>
            <p className="mt-2 text-muted-foreground">Redirecting to home...</p>
          </>
        )}
        
        {status === "error" && (
          <>
            <div className="mb-4 text-6xl">✗</div>
            <h1 className="text-2xl font-bold text-red-600">Login Failed</h1>
            <p className="mt-2 text-muted-foreground">{error}</p>
            <button
              onClick={() => router.push("/")}
              className="mt-4 rounded bg-primary px-4 py-2 text-primary-foreground hover:bg-primary/90"
            >
              Return to Home
            </button>
          </>
        )}
      </div>
    </div>
  );
}
