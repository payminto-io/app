import { Suspense } from "react";
import { SigninForm } from "./signin-form";

export const dynamic = "force-dynamic";

export default function SigninPage() {
  return (
    <Suspense fallback={null}>
      <SigninForm />
    </Suspense>
  );
}
