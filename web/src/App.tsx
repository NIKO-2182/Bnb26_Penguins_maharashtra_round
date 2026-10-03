/**
 * File: web/src/App.tsx
 * Purpose: Route table. "/" is the user view, "/operator" the dashboard.
 */
import { lazy, Suspense } from "react"
import { BrowserRouter, Navigate, Route, Routes } from "react-router-dom"
import UserView from "@/pages/UserView"

const Operator = lazy(() => import("@/pages/Operator"))

export default function App() {
  return (
    <BrowserRouter>
      <Suspense fallback={null}>
        <Routes>
          <Route path="/" element={<UserView />} />
          <Route path="/operator" element={<Operator />} />
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </Suspense>
    </BrowserRouter>
  )
}
