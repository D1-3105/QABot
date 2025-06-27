import { useState } from 'react'
import { BrowserRouter, Routes, Route } from "react-router-dom";
import JobLogsPage from "./JobLogsPage";
import './App.css'

function App() {

    return (
        <BrowserRouter>
            <Routes>
                <Route path="/job/logs" element={<JobLogsPage />} />
            </Routes>
        </BrowserRouter>
    );
}

export default App
