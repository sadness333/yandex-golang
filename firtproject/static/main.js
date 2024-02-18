
var inputValues = {
    input1: '',
    input2: '',
    input3: '',
    input4: '',
    input5: ''
};

function switchContent(contentId) {
    document.getElementById('calculatorContent').style.display = 'none';
    document.getElementById('settingsContent').style.display = 'none';
    document.getElementById('resourceContent').style.display = 'none';

    document.getElementById(contentId + 'Content').style.display = 'block';
}

document.getElementById("expressionInput").addEventListener("keypress", function(event) {
    if (event.key === "Enter") {
        event.preventDefault(); 
        submitForm();
    }
});

function submitForm() {
    var expressionValue = document.getElementById("expressionInput").value;
    var requestId = generateUUID(); 
    document.getElementById("expressionInput").value = ""
    console.log("Expression submitted with requestId: " + requestId);

    // Создаем объект с данными для отправки
    var requestData = {
        expression: expressionValue,
        requestId: requestId
    };

    fetch("/addExpression", {
        method: "POST",
        headers: {
            "Content-Type": "application/json"
        },
        body: JSON.stringify(requestData)
    })
    .then(response => {
        if (response.ok) {
            console.log("Request successfully processed.");

            // Обновление фронтенда после успешного обновления результата
            getLatestTasks();  // Вызываем функцию получения последних задач
        } else {
            console.error("Error processing the request.");
        }
    })
    .catch(error => {
        console.error("Error:", error);
    });
}

function generateUUID() {
    return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, function(c) {
        var r = Math.random() * 16 | 0,
            v = c === 'x' ? r : (r & 0x3 | 0x8);
        return v.toString(16);
    });
}

function validateInput(input) {
    // Оставляем только цифры в значении инпута
    input.value = input.value.replace(/[^0-9]/g, '');
}



function getLatestTasks() {
    fetch("/getExpressions") 
        .then(response => response.json())
        .then(tasks => {
            // Ограничиваем количество записей до 10
            const lastTenTasks = tasks.slice(0, 10);
            displayTasks(lastTenTasks);
        })
        .catch(error => {
            console.error("Error fetching tasks:", error);
        });
}

function displayTasks(tasks) {
    var taskList = document.getElementById("taskList");

    // Очищаем содержимое taskList перед добавлением новых записей
    taskList.innerHTML = "";

    tasks.forEach(task => {
        var listItem = document.createElement("li");

        // Check if the status is "completed" to display the result
        var resultText = 'Result: не готово';
        if (task.status === "completed") {
            if (task.result !== undefined && task.result !== null && task.result.Float64 !== undefined) {
                resultText = `Result: ${task.result.Float64}`;
            } else {
                resultText = 'Result: данные отсутствуют';
            }
        }

        listItem.textContent = `Expression: ${task.expression}, Status: ${task.status}, ${resultText}`;
        taskList.appendChild(listItem);
    });
}

document.addEventListener("DOMContentLoaded", function() {
    // Вызываем функцию, когда документ загружен
    getLatestTasks();
});


document.addEventListener('DOMContentLoaded', function() {
    // Выполнение запроса на /getOperations
    fetch('/getOperations')
        .then(response => {
            if (!response.ok) {
                throw new Error('Network response was not ok');
            }
            return response.json();
        })
        .then(data => {
            // Заполнение инпутов данными из полученного объекта
            document.getElementById('input1').value = parseInt(data.addition) || '';
            document.getElementById('input2').value = parseInt(data.subtraction) || '';
            document.getElementById('input3').value = parseInt(data.multiplication) || '';
            document.getElementById('input4').value = parseInt(data.division) || '';
            document.getElementById('input5').value = parseInt(data.server_idle) || '';
        })
        .catch(error => console.error('Error:', error));
});

function applyChanges() {
    // Получение значений из инпутов
    var input1Value = document.getElementById('input1').value;
    var input2Value = document.getElementById('input2').value;
    var input3Value = document.getElementById('input3').value;
    var input4Value = document.getElementById('input4').value;
    var input5Value = document.getElementById('input5').value;

    // Создание объекта с данными
    var requestData = {
        input1: parseInt(input1Value),
        input2: parseInt(input2Value),
        input3: parseInt(input3Value),
        input4: parseInt(input4Value),
        input5: parseInt(input5Value)
    };

    // Отправка данных на бэкенд
    fetch("/getOperation", {
        method: "POST",
        headers: {
            "Content-Type": "application/json"
        },
        body: JSON.stringify(requestData)
    })
    .then(response => {
        if (response.ok) {
            console.log("Changes successfully applied.");
        } else {
            console.error("Error applying changes.");
        }
    })
    .catch(error => {
        console.error("Error:", error);
    });
}