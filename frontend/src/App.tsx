import { NavLink, Route, Routes } from 'react-router-dom'
import CheckoutPage from './pages/CheckoutPage'
import ReportPage from './pages/ReportPage'
import ReturnPage from './pages/ReturnPage'

export default function App() {
  return (
    <div className="app-shell">
      <header className="topbar">
        <NavLink to="/" className="brand"><span className="brand-mark">A</span> Arc Pay</NavLink>
        <nav aria-label="Main navigation">
          <NavLink to="/" end>Payment</NavLink>
          <NavLink to="/report">Report</NavLink>
        </nav>
      </header>
      <main>
        <Routes>
          <Route path="/" element={<CheckoutPage />} />
          <Route path="/report" element={<ReportPage />} />
          <Route path="/payment-complete" element={<ReturnPage />} />
        </Routes>
      </main>
      <footer>Powered by Stripe · Card details are handled securely by Stripe</footer>
    </div>
  )
}

