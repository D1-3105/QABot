import { BrowserRouter, Routes, Route } from "react-router-dom";
import JobLogsPage from "./JobLogsPage";
import './App.css'
import MainPage from "./MainPage.js";

function App() {

    return (
        <BrowserRouter>
            <Routes>
                <Route path="/" element={<MainPage />}/>
                <Route path="/job/logs" element={<JobLogsPage />} />
            </Routes>
        </BrowserRouter>
    );
}

export default App
