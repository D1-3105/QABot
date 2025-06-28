import React, { useEffect, useState } from "react"
import ReactMarkdown from "react-markdown"

const MainPage = () => {
    const [body, setBody] = useState("")
    const [error, setError] = useState(null)

    useEffect(() => {
        fetch("/api/v1/help")
            .then((res) => {
                if (!res.ok) throw new Error(`HTTP error! status: ${res.status}`)
                return res.json()
            })
            .then((jsBody) => setBody(jsBody.body))
            .catch((err) => setError(err.message))
    }, [])

    if (error) return <div style={{ textAlign: "left", padding: "1rem" }}>Error: {error}</div>

    return (
        <div style={{ textAlign: "left", padding: "1rem" }}>
            <a href="/swagger/index.html"><h2>Swagger url</h2></a>
            <h2>Help:</h2>
            <ReactMarkdown>{body}</ReactMarkdown>
        </div>
    )
}

export default MainPage
